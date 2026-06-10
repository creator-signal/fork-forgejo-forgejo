package lfs

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	git_model "forgejo.org/models/git"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	quota_model "forgejo.org/models/quota"
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

type OidLine struct {
	Oid  string
	Size int
	Args map[string]string
}

type SSHAdpater struct {
	ctx           context.Context
	user          *user_model.User
	repository    *repo_model.Repository
	requestedMode perm.AccessMode
	pktAdapter    *PktAdapter
}

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

func parseBatchRequest(req [][]byte) ([]OidLine, error) {
	var oidLines []OidLine
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
		args := make(map[string]string)
		if len(values) > 2 {
			for _, value := range values[2:] {
				key, value, found := strings.Cut(strings.TrimRight(value, "\n"), "=")
				if !found {
					return oidLines, fmt.Errorf("Error parsing batch request arguments %q: expected \"key=value\"", value)
				}
				args[key] = value
			}
		}
		oidLines = append(oidLines,
			OidLine{
				Oid:  values[0],
				Size: size,
				Args: args,
			})
	}
	return oidLines, nil
}

func (s *SSHAdpater) getCapabilityAdvertisement() []byte {
	return []byte("version=1\n")
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

func (s *SSHAdpater) handleBatchRequest(lfsVerb string) (PktLine, int, error) {
	var batchOidLine PktLine
	packets, err := s.pktAdapter.Read()
	if err != nil {
		return batchOidLine, http.StatusBadRequest, errors.New("Failed to read LFS command")
	}

	oidLines, err := parseBatchRequest(packets)
	if err != nil {
		return batchOidLine, http.StatusBadRequest, fmt.Errorf("Error parsing batch request: %v", err)
	}

	for _, oidLine := range oidLines {
		batchLine, err := NewPktLine(fmt.Appendf(nil, "%s %d %s", oidLine.Oid, oidLine.Size, lfsVerb))
		if err != nil {
			return batchOidLine, http.StatusInternalServerError, fmt.Errorf("Error creating batch response: %v", err)
		}
		batchOidLine = append(batchOidLine, batchLine...)
	}
	return batchOidLine, http.StatusOK, nil
}

func (s *SSHAdpater) handlePutObjectRequest(command string, packets [][]byte) (int, error) {
	if s.requestedMode < perm.AccessModeWrite {
		return http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for put-object command", s.requestedMode)
	}
	_, oid, found := strings.Cut(command, " ")
	if !found {
		return http.StatusUnprocessableEntity, fmt.Errorf("Cannot find oid in put-object request: %q", command)
	}
	if len(packets) <= 1 {
		return http.StatusBadRequest, errors.New("Put-object request must have more than 1 packets")
	}
	args, err := parseArguments(packets[1:])
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("Error parsing put-object arguments: %v", err)
	}
	size, ok := args["size"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("Size argument not found in put-object: %q %q", packets, args)
	}

	p := lfs_module.Pointer{Oid: oid}
	if p.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return http.StatusUnprocessableEntity, fmt.Errorf("Invalid size: %q", size)
	}
	if !p.IsValid() {
		return http.StatusUnprocessableEntity, fmt.Errorf("Attempt to access invalid LFS OID[%s] in %s", p.Oid, s.repository.Name)
	}

	binaryReader := s.pktAdapter.GetBinaryReader()
	contentStore := lfs_module.NewContentStore()
	exists, err := contentStore.Exists(p)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Unable to check if LFS OID[%s] exist. Error: %v", p.Oid, err)
	}

	if exists {
		ok, err := quota_model.EvaluateForUser(s.ctx, s.user.ID, quota_model.LimitSubjectSizeGitLFS)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("quota_model.EvaluateForUser: %v", err)
		}
		if !ok {
			return http.StatusRequestEntityTooLarge, errors.New("quota exceeded")
		}
	}

	uploadOrVerify := func() error {
		if exists {
			accessible, err := git_model.LFSObjectAccessible(s.ctx, s.user, p.Oid)
			if err != nil {
				return fmt.Errorf("Unable to check if LFS MetaObject [%s] is accessible. Error: %v", p.Oid, err)
			}
			if !accessible {
				// The file exists but the user has no access to it.
				// The upload gets verified by hashing and size comparison to prove access to it.
				hash := sha256.New()
				written, err := io.Copy(hash, binaryReader)
				if err != nil {
					return fmt.Errorf("Error creating hash. Error: %v", err)
				}

				if written != p.Size {
					return fmt.Errorf("content size does not match (got %d, expected %d)", written, p.Size)
				}
				if hex.EncodeToString(hash.Sum(nil)) != p.Oid {
					return lfs_module.ErrHashMismatch
				}
			} else {
				_, err := io.Copy(io.Discard, binaryReader) // Discarding data
				if err != nil {
					return fmt.Errorf("Error discarding data: %v", err)
				}
			}
		} else if err := contentStore.Put(p, binaryReader); err != nil {
			return fmt.Errorf("error copying data: %v", err)
		}
		_, err := git_model.NewLFSMetaObject(s.ctx, s.repository.ID, p)
		return err
	}

	if err := uploadOrVerify(); err != nil {
		returnErr := fmt.Errorf("Error whilst uploadOrVerify LFS OID[%s]: %v", p.Oid, err)
		if _, err = git_model.RemoveLFSMetaObjectByOid(s.ctx, s.repository.ID, p.Oid); err != nil {
			returnErr = errors.Join(fmt.Errorf("Error whilst removing MetaObject for LFS OID[%s]: %v", p.Oid, err))
		}
		return http.StatusInternalServerError, returnErr
	}
	return http.StatusOK, nil
}

func (s *SSHAdpater) handleVerifyObject(command string, packets [][]byte) (int, error) {
	if s.requestedMode < perm.AccessModeRead {
		return http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for verify-object command", s.requestedMode)
	}
	_, oid, found := strings.Cut(command, " ")
	if !found {
		return http.StatusUnprocessableEntity, fmt.Errorf("Cannot find oid in verify-object request: %q", command)
	}
	if len(packets) <= 1 {
		return http.StatusBadRequest, errors.New("Verify-object request must have more than 1 packets")
	}
	args, err := parseArguments(packets[1:])
	if err != nil {
		return http.StatusBadRequest, fmt.Errorf("Error parsing verify-object arguments: %v", err)
	}
	size, ok := args["size"]
	if !ok {
		return http.StatusBadRequest, fmt.Errorf("Size argument not found in verify-object: %q %q", packets, args)
	}

	p := lfs_module.Pointer{Oid: oid}
	if p.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return http.StatusUnprocessableEntity, fmt.Errorf("Invalid size: %q", size)
	}
	contentStore := lfs_module.NewContentStore()
	ok, err = contentStore.Verify(p)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error whilst verifying LFS OID[%s]: %v", p.Oid, err)
	} else if !ok {
		return http.StatusNotFound, fmt.Errorf("LFS OID[%s] not found", p.Oid)
	}
	return http.StatusOK, nil
}

func (s *SSHAdpater) handleGetObjectRequest(command string) (int, error) {
	if s.requestedMode < perm.AccessModeRead {
		return http.StatusUnauthorized, fmt.Errorf("Requested mode %s does not allow for get-object command", s.requestedMode)
	}
	_, oid, found := strings.Cut(command, " ")
	if !found {
		return http.StatusUnprocessableEntity, fmt.Errorf("Cannot find oid in get-object request: %q", command)
	}

	p := lfs_module.Pointer{Oid: oid}
	if !p.IsValid() {
		return http.StatusUnprocessableEntity, errors.New("Oid or size are invalid")
	}
	meta, err := git_model.GetLFSMetaObjectByOid(s.ctx, s.repository.ID, p.Oid)
	if err != nil {
		return http.StatusNotFound, errors.New("Unable to get LFS oid")
	}

	contentStore := lfs_module.NewContentStore()
	content, err := contentStore.Get(meta.Pointer)
	if err != nil {
		return http.StatusNotFound, errors.New("Content not found")
	}
	defer content.Close()
	err = s.pktAdapter.WriteBinaryData(content)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error whilst sending LFS OID[%s] data: %v", p.Oid, err)
	}

	return http.StatusOK, nil
}

func HandleLFSTransfer(ctx context.Context, results *private.ServCommandResults, pktAdapter *PktAdapter,
	requestedMode perm.AccessMode, lfsVerb string,
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
	sshAdapter := SSHAdpater{ctx: ctx, user: user, repository: repository, requestedMode: requestedMode, pktAdapter: pktAdapter}
	if !sshAdapter.CheckAuthorization() {
		return fmt.Errorf("Not authorized to access repository")
	}

	err = pktAdapter.WriteData(sshAdapter.getCapabilityAdvertisement())
	if err != nil {
		return fmt.Errorf("Failed to send capability advertisement: %v", err)
	}
	err = pktAdapter.WriteFlush()
	if err != nil {
		return fmt.Errorf("Failed to flush capability advertisement: %v", err)
	}

	packets, err := pktAdapter.Read()
	if err != nil {
		return fmt.Errorf("Failed to read capability response: %v", err)
	} else if len(packets) != 1 {
		return fmt.Errorf("Unexpected capability response: more than 1 packet received")
	}

	quitExpected := false
	var finalErr error
	if !sshAdapter.checkVersionCommand(packets[0]) {
		err = pktAdapter.WriteHTTPError(400, "Unexpected version received")
		if err != nil {
			return fmt.Errorf("Failed to send version error: %v", err)
		}
		return fmt.Errorf("Failed to match capability: %q", packets[0])
	}
	err = pktAdapter.WriteHTTPOK()
	if err != nil {
		return fmt.Errorf("Failed to send version acknowledgment: %v", err)
	}

	for {
		packets, err = pktAdapter.Read()
		if err != nil {
			statusErr := pktAdapter.WriteHTTPError(400, "Failed to read LFS command")
			if statusErr != nil {
				err = errors.Join(err, statusErr)
			}
			return fmt.Errorf("Failed to read LFS command: %v", err)
		} else if len(packets) == 0 {
			statusErr := pktAdapter.WriteHTTPError(400, "Unexpected empty LFS command")
			if statusErr != nil {
				err = errors.Join(err, statusErr)
			}
			return fmt.Errorf("Unexpected empty LFS command: %v", err)
		}

		command := strings.TrimRight(string(packets[0]), "\n")
		if command == "quit" {
			quitErr := pktAdapter.WriteHTTPOK()
			if quitErr != nil {
				if finalErr != nil {
					finalErr = errors.Join(finalErr, quitErr)
				} else {
					finalErr = quitErr
				}
			}
			break
		} else if quitExpected {
			return fmt.Errorf("Unexpected LFS command waiting for quit: %q", packets)
		}

		if command == "batch" {
			batchOidLine, status, err := sshAdapter.handleBatchRequest(lfsVerb)
			if err != nil {
				statusErr := pktAdapter.WriteHTTPError(status, err.Error())
				if statusErr != nil {
					return fmt.Errorf("Failed reporting error during batch request handling: %v", errors.Join(err, statusErr))
				}
				finalErr = err
				quitExpected = true
				continue
			}
			statusPkt, err := NewPktLine(fmt.Appendf(nil, "status %d\n", 200))
			if err != nil {
				return fmt.Errorf("Error creating status PktLine: %v", err)
			}
			if err := pktAdapter.WriteSplitPacket(statusPkt, batchOidLine); err != nil {
				return fmt.Errorf("Error sending batch response: %v", err)
			}
		} else if strings.HasPrefix(command, "put-object") {
			status, err := sshAdapter.handlePutObjectRequest(command, packets)
			if err != nil {
				statusErr := pktAdapter.WriteHTTPError(status, err.Error())
				if statusErr != nil {
					return fmt.Errorf("Failed error reporting for put-object request handling: %v", errors.Join(err, statusErr))
				}
				finalErr = err
				quitExpected = true
				continue
			}
			if err := pktAdapter.WriteHTTPOK(); err != nil {
				return fmt.Errorf("Error sending OK response to put-object: %v", err)
			}
		} else if strings.HasPrefix(command, "verify-object") {
			status, err := sshAdapter.handleVerifyObject(command, packets)
			if err != nil {
				statusErr := pktAdapter.WriteHTTPError(status, err.Error())
				if statusErr != nil {
					return fmt.Errorf("Failed error reporting for put-object request handling: %v", errors.Join(err, statusErr))
				}
				finalErr = err
				quitExpected = true
				continue
			}
			if err := pktAdapter.WriteHTTPOK(); err != nil {
				return fmt.Errorf("Error sending OK response to put-object: %v", err)
			}
		} else if strings.HasPrefix(command, "get-object") {
			statusCode, err := sshAdapter.handleGetObjectRequest(command)
			if err != nil {
				err := pktAdapter.WriteHTTPError(statusCode, err.Error())
				if err != nil {
					return fmt.Errorf("Error sending error status: %v", err)
				}
				quitExpected = true
				continue
			}
		} else {
			if err := pktAdapter.WriteHTTPError(400, fmt.Sprintf("Unrecognized command: %q", command)); err != nil {
				return fmt.Errorf("Unrecognized LFS command: %v", err)
			}
			quitExpected = true
		}
	}
	if finalErr != nil {
		return fmt.Errorf("Unexpected error during LFS transfer: %v", err)
	}
	return nil
}
