// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package lfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	lfs_module "forgejo.org/modules/lfs"
	"forgejo.org/modules/log"
	"forgejo.org/modules/private"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
)

// See https://github.com/git-lfs/git-lfs/blob/main/docs/proposals/ssh_adapter.md

const (
	batchCommand        = "batch"
	getObjectCommand    = "get-object"
	listLockCommand     = "list-lock"
	lockCommand         = "lock"
	putObjectCommand    = "put-object"
	unlockCommand       = "unlock"
	verifyObjectCommand = "verify-object"
)

type OidLine struct {
	Oid  string
	Size int
	Args map[string]string
}

type SSHAdpater struct {
	bridge        *HTTPBridge
	client        *lfs_module.Client
	ctx           context.Context
	handlerMap    map[string]commandHandler
	lfsVerb       string
	pktAdapter    *PktAdapter
	quitExpected  bool
	repository    *repo_model.Repository
	requestedMode perm.AccessMode
	user          *user_model.User
}

type commandHandler func(string, [][]byte) error

func parseArguments(req [][]byte) (map[string]string, error) {
	arguments := make(map[string]string)
	for _, packet := range req {
		if packet == nil {
			break
		}
		key, value, found := strings.Cut(strings.TrimRight(string(packet), "\n"), "=")
		if !found {
			return arguments, fmt.Errorf("Error parsing batch request arguments %q: expected \"key=value\"", packet)
		}
		arguments[key] = value
	}
	return arguments, nil
}

func parseBatchRequest(req [][]byte) ([]lfs_module.Pointer, error) {
	var oidLines []lfs_module.Pointer
	for _, packet := range req {
		if packet == nil {
			break
		}
		values := strings.Split(strings.TrimRight(string(packet), "\n"), " ")
		if len(values) < 2 {
			return oidLines, fmt.Errorf("Error parsing batch request oid %q: expected \"oid size *[key=value]\"", string(packet))
		}
		size, err := strconv.Atoi(values[1])
		if err != nil {
			return oidLines, fmt.Errorf("Error parsing batch request oid's size: %q", values[1])
		}
		oidLines = append(oidLines,
			lfs_module.Pointer{
				Oid:  values[0],
				Size: int64(size),
			})
	}
	return oidLines, nil
}

func (s *SSHAdpater) checkVersionCommand(c []byte) bool {
	return bytes.Equal(c, []byte("version 1\n"))
}

func (s *SSHAdpater) CheckAuthorization() bool {
	perm, err := access_model.GetUserRepoPermission(s.ctx, s.repository, s.user)
	if err != nil {
		log.Error("Unable to GetUserRepoPermission for user %-v in repo %-v Error: %v", s.user, s.repository, err)
		return false
	}

	return perm.CanAccess(s.requestedMode, unit.TypeCode)
}

func (s *SSHAdpater) handleError(status int, err error) error {
	reportError := s.pktAdapter.WriteHTTPError(status, err.Error())
	if err != nil {
		return fmt.Errorf("Error sending error (status %d: %v): %v", status, err, reportError)
	}
	s.quitExpected = true
	return nil
}

func (s *SSHAdpater) handleBatchRequest(_ string, _ [][]byte) error {
	packets, err := s.pktAdapter.Read()
	if err != nil {
		return s.handleError(http.StatusBadRequest, errors.New("Failed to read batch request data"))
	}

	oidLines, err := parseBatchRequest(packets)
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing batch request: %v", err))
	}
	err = s.bridge.Batch(s.ctx, s.user, s.repository, s.lfsVerb, oidLines)
	if err != nil {
		return s.handleError(http.StatusInternalServerError, err)
	}

	var batchOidLine PktLine
	for _, oidLine := range oidLines {
		batchLine, err := NewPktLine(fmt.Appendf(nil, "%s %d %s", oidLine.Oid, oidLine.Size, s.lfsVerb))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("Error creating batch response: %v", err))
		}
		batchOidLine = append(batchOidLine, batchLine...)
	}

	statusPkt, err := NewPktLine(fmt.Appendf(nil, "status %d\n", http.StatusOK))
	if err != nil {
		return fmt.Errorf("Error creating status PktLine: %v", err)
	}
	if err := s.pktAdapter.WriteSplitPacket(statusPkt, batchOidLine); err != nil {
		return fmt.Errorf("Error sending batch response: %v", err)
	}
	return nil
}

func (s *SSHAdpater) handlePutObjectRequest(oid string, packets [][]byte) error {
	if s.requestedMode < perm.AccessModeWrite {
		return s.handleError(
			http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for put-object command", s.requestedMode))
	}
	if oid == "" {
		return s.handleError(http.StatusUnprocessableEntity, errors.New("No oid in put-object request"))
	}
	if len(packets) <= 1 {
		return s.handleError(http.StatusBadRequest, errors.New("Put-object request must have more than 1 packets"))
	}
	args, err := parseArguments(packets[1:])
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing put-object arguments: %v", err))
	}
	size, ok := args["size"]
	if !ok {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Size argument not found in put-object: %q %q", packets, args))
	}

	pointer := &lfs_module.Pointer{Oid: oid}
	if pointer.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return s.handleError(http.StatusUnprocessableEntity, fmt.Errorf("Invalid size: %q", size))
	}
	if !pointer.IsValid() {
		return s.handleError(
			http.StatusUnprocessableEntity, fmt.Errorf("Attempt to access invalid LFS OID[%s] in %s", pointer.Oid, s.repository.Name))
	}

	err = s.bridge.Upload(s.ctx, s.user, s.repository, pointer, s.pktAdapter)
	if err != nil {
		return s.handleError(http.StatusInternalServerError, err)
	}

	if err := s.pktAdapter.WriteHTTPOK(); err != nil {
		return fmt.Errorf("Error sending OK response to put-object: %v", err)
	}
	return nil
}

func (s *SSHAdpater) handleVerifyObject(oid string, packets [][]byte) error {
	if s.requestedMode < perm.AccessModeRead {
		return s.handleError(
			http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for verify-object command", s.requestedMode))
	}
	if oid == "" {
		return s.handleError(http.StatusUnprocessableEntity, errors.New("No oid in verify-object"))
	}
	if len(packets) <= 1 {
		return s.handleError(http.StatusBadRequest, errors.New("Verify-object request must have more than 1 packets"))
	}
	args, err := parseArguments(packets[1:])
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing verify-object arguments: %v", err))
	}
	sizeStr, ok := args["size"]
	if !ok {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Size argument not found in verify-object: %q %q", packets, args))
	}
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return s.handleError(
			http.StatusBadRequest, fmt.Errorf("Size argument (%s) cannot be parsed as an integer in verify-object: %v", sizeStr, err))
	}

	pointer := &lfs_module.Pointer{Oid: oid, Size: size}
	err = s.bridge.Verify(s.ctx, s.user, s.repository, pointer)
	if err != nil {
		return s.handleError(http.StatusInternalServerError, err)
	}

	if err := s.pktAdapter.WriteHTTPOK(); err != nil {
		return fmt.Errorf("Error sending OK response to put-object: %v", err)
	}
	return nil
}

func (s *SSHAdpater) handleGetObjectRequest(oid string, _ [][]byte) error {
	if s.requestedMode < perm.AccessModeRead {
		return s.handleError(
			http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for get-object command", s.requestedMode))
	}
	if oid == "" {
		return s.handleError(http.StatusUnprocessableEntity, errors.New("No oid in get-object request"))
	}

	pointer := &lfs_module.Pointer{Oid: oid}
	if !pointer.IsValid() {
		return s.handleError(http.StatusUnprocessableEntity, errors.New("Oid or size are invalid"))
	}

	err := s.bridge.Download(s.ctx, s.user, s.repository, pointer, s.pktAdapter)
	if err != nil {
		return s.handleError(http.StatusInternalServerError, err)
	}
	return nil
}

func (s *SSHAdpater) handleListLockRequest(_ string, packets [][]byte) error {
	requestArgs, err := parseArguments(packets[1:])
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing list-lock arguments: %v", err))
	}

	var lockSpec PktLine
	lockList, err := s.bridge.ListLock(s.ctx, s.user, s.repository, requestArgs)
	if err != nil {
		return s.handleError(http.StatusInternalServerError, fmt.Errorf("error requesting lock list from bridge: %v", err))
	}

	for _, lock := range lockList.Locks {
		idLine, err := NewPktLine(fmt.Appendf(nil, "lock %s", lock.ID))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("error creating PktLine for lock[%s] id", lock.ID))
		}
		pathLine, err := NewPktLine(fmt.Appendf(nil, "path %s %s", lock.ID, lock.Path))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("error creating PktLine for lock[%s] patch", lock.ID))
		}
		atLine, err := NewPktLine(fmt.Appendf(nil, "locked-at %s %s", lock.ID, lock.LockedAt.Format(time.RFC3339)))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("error creating PktLine for lock[%s] locked-at", lock.ID))
		}
		ownerNameLine, err := NewPktLine(fmt.Appendf(nil, "ownername %s %s", lock.ID, lock.Owner.Name))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("error creating PktLine for lock[%s] ownername", lock.ID))
		}
		var owning string
		if lock.Owner.Name == s.user.Name {
			owning = "ours"
		} else {
			owning = "theirs"
		}
		ownerLine, err := NewPktLine(fmt.Appendf(nil, "owner %s %s", lock.ID, owning))
		if err != nil {
			return s.handleError(http.StatusInternalServerError, fmt.Errorf("error creating PktLine for lock[%s] owner", lock.ID))
		}
		lockSpec = slices.Concat(lockSpec, idLine, pathLine, atLine, ownerNameLine, ownerLine)
	}
	statusPkt, err := NewPktLine(fmt.Appendf(nil, "status %d\n", http.StatusOK))
	if err != nil {
		return fmt.Errorf("Error creating status PktLine: %v", err)
	}
	if err := s.pktAdapter.WriteSplitPacket(statusPkt, lockSpec); err != nil {
		return fmt.Errorf("Error sending list-lock response: %v", err)
	}
	return nil
}

func (s *SSHAdpater) handleLockRequest(_ string, packets [][]byte) error {
	args, err := parseArguments(packets[1:])
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing lock arguments: %v", err))
	}
	path, ok := args["path"]
	if !ok {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Path argument not found in lock: %q %q", packets, args))
	}

	lockResponse, errorResponse, code, ok, err := s.bridge.Lock(s.ctx, s.user, s.repository, path)
	if err != nil {
		return s.handleError(code, err)
	}

	if !ok {
		return s.pktAdapter.WriteHTTPErrorWithArgs(code, errorResponse.Message, map[string]any{
			"id": errorResponse.Lock.ID, "path": errorResponse.Lock.Path, "locked-at": errorResponse.Lock.LockedAt,
			"ownername": s.user.Name,
		})
	}
	return s.pktAdapter.WriteStatusWithArgs(code, map[string]any{
		"id": lockResponse.Lock.ID, "path": lockResponse.Lock.Path, "locked-at": lockResponse.Lock.LockedAt,
		"ownername": s.user.Name,
	})
}

func (s *SSHAdpater) handleUnlockRequest(lid string, packets [][]byte) error {
	if lid == "" {
		return s.handleError(http.StatusUnprocessableEntity, errors.New("No lock id in unlock request"))
	}
	if len(packets) <= 1 {
		return s.handleError(http.StatusBadRequest, errors.New("Unlock request must have more than 1 packet"))
	}
	args, err := parseArguments(packets[1:])
	if err != nil {
		return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing unlock arguments: %v", err))
	}
	forceStr, ok := args["force"]
	force := false
	if ok {
		force, err = strconv.ParseBool(forceStr)
		if err != nil {
			return s.handleError(http.StatusBadRequest, fmt.Errorf("Error parsing unlock force argument (%s): %v", forceStr, err))
		}
	}

	lockResponse, errorResponse, code, ok, err := s.bridge.Unlock(s.ctx, s.user, s.repository, lid, force)
	if err != nil {
		return s.handleError(code, err)
	}

	if !ok {
		return s.pktAdapter.WriteHTTPErrorWithArgs(code, errorResponse.Message, map[string]any{
			"id": errorResponse.Lock.ID, "path": errorResponse.Lock.Path, "locked-at": errorResponse.Lock.LockedAt,
			"ownername": s.user.Name,
		})
	}
	return s.pktAdapter.WriteStatusWithArgs(code, map[string]any{
		"id": lockResponse.Lock.ID, "path": lockResponse.Lock.Path, "locked-at": lockResponse.Lock.LockedAt,
		"ownername": s.user.Name,
	})
}

func (s *SSHAdpater) init() error {
	s.handlerMap = map[string]commandHandler{
		batchCommand:        s.handleBatchRequest,
		getObjectCommand:    s.handleGetObjectRequest,
		listLockCommand:     s.handleListLockRequest,
		lockCommand:         s.handleLockRequest,
		putObjectCommand:    s.handlePutObjectRequest,
		unlockCommand:       s.handleUnlockRequest,
		verifyObjectCommand: s.handleVerifyObject,
	}

	if !s.CheckAuthorization() {
		return fmt.Errorf("Not authorized to access repository")
	}

	err := s.pktAdapter.WriteStr("version=1")
	if err != nil {
		return fmt.Errorf("Failed to send version capability advertisement: %v", err)
	}
	err = s.pktAdapter.WriteStr("locking")
	if err != nil {
		return fmt.Errorf("Failed to send locking capability advertisement: %v", err)
	}
	err = s.pktAdapter.WriteFlush()
	if err != nil {
		return fmt.Errorf("Failed to flush capability advertisement: %v", err)
	}
	return nil
}

func (s *SSHAdpater) run() error {
	err := s.init()
	if err != nil {
		return err
	}

	packets, err := s.pktAdapter.Read()
	if err != nil {
		return fmt.Errorf("Failed to read capability response: %v", err)
	} else if len(packets) != 1 {
		return fmt.Errorf("Unexpected capability response: more than 1 packet received")
	}

	if !s.checkVersionCommand(packets[0]) {
		err = s.pktAdapter.WriteHTTPError(http.StatusBadRequest, "Unexpected version received")
		if err != nil {
			return fmt.Errorf("Failed to send version error: %v", err)
		}
		return fmt.Errorf("Failed to match capability: %q", packets[0])
	}
	err = s.pktAdapter.WriteHTTPOK()
	if err != nil {
		return fmt.Errorf("Failed to send version acknowledgment: %v", err)
	}

	for {
		packets, err = s.pktAdapter.Read()
		if err != nil {
			if err == io.EOF { // client closed connection
				return nil
			}
			statusErr := s.pktAdapter.WriteHTTPError(http.StatusInternalServerError, "Failed to read LFS command")
			if statusErr != nil {
				err = errors.Join(err, statusErr)
			}
			return fmt.Errorf("Failed to read LFS command: %v", err)
		} else if len(packets) == 0 {
			statusErr := s.pktAdapter.WriteHTTPError(http.StatusBadRequest, "Unexpected empty LFS command")
			if statusErr != nil {
				err = errors.Join(err, statusErr)
			}
			return fmt.Errorf("Unexpected empty LFS command: %v", err)
		}

		commandName, commandArg, _ := strings.Cut(strings.TrimRight(string(packets[0]), "\n"), " ")
		if commandName == "quit" {
			err = s.pktAdapter.WriteHTTPOK()
			break
		} else if s.quitExpected {
			return fmt.Errorf("Unexpected LFS command waiting for quit: %q", packets)
		}

		handler, found := s.handlerMap[commandName]
		if !found {
			if err := s.pktAdapter.WriteHTTPError(http.StatusBadRequest, fmt.Sprintf("Unrecognized command: %q", commandName)); err != nil {
				return fmt.Errorf("Unrecognized LFS command: %v", err)
			}
			s.quitExpected = true
		} else {
			err = handler(commandArg, packets)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		return fmt.Errorf("Unexpected error during LFS transfer: %v", err)
	}
	return nil
}

func HandleLFSTransfer(ctx context.Context, results *private.ServCommandResults, pktAdapter *PktAdapter,
	requestedMode perm.AccessMode, lfsVerb, tokenString string,
) error {
	if !setting.LFS.StartServer {
		return fmt.Errorf("LFS isn't enabled")
	}
	if err := storage.InitLFS(); err != nil {
		return fmt.Errorf("Cannot initialize LFS storage: %v", err)
	}

	user, err := user_model.GetUserByID(ctx, results.UserID)
	if err != nil {
		return fmt.Errorf("Unable to GetUserById: %v", err)
	}
	repository, err := repo_model.GetRepositoryByOwnerAndName(ctx, results.OwnerName, results.RepoName)
	if err != nil {
		return fmt.Errorf("You must have pull access to %s/%s: %v", results.OwnerName, results.RepoName, err)
	}
	localURL, err := url.Parse(setting.LocalURL)
	if err != nil {
		return fmt.Errorf("Cannot create local url for LFS HTTP bridge: %v", err)
	}
	client := lfs_module.NewClient(localURL, nil)
	sshAdapter := SSHAdpater{
		bridge: newHTTPBridge(tokenString), client: &client, ctx: ctx, lfsVerb: lfsVerb, pktAdapter: pktAdapter,
		repository: repository, requestedMode: requestedMode, user: user,
	}
	return sshAdapter.run()
}
