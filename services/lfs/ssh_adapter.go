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
)

// See https://github.com/git-lfs/git-lfs/blob/main/docs/proposals/ssh_adapter.md

type OidLine struct {
	Oid  string
	Size int
	Args map[string]string
}

func GetCapabilityAdvertisement() []byte {
	return []byte("version=1\n")
}

func CheckVersionCommand(c []byte) bool {
	return bytes.Equal(c, []byte("version 1\n"))
}

func CheckAuthorization(ctx context.Context, user *user_model.User, repository *repo_model.Repository, accessMode perm.AccessMode) bool {
	perm, err := access_model.GetUserRepoPermission(ctx, repository, user)
	if err != nil {
		log.Error("Unable to GetUserRepoPermission for user %-v in repo %-v Error: %v", user, repository, err)
		return false
	}

	return perm.CanAccess(accessMode, unit.TypeCode)
}

func ParseArguments(req [][]byte) (map[string]string, error) {
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

func ParseBatchRequest(req [][]byte) ([]OidLine, error) {
	var oidLines []OidLine
	for _, packet := range req {
		if packet == nil {
			break
		}
		packetStr := string(packet[:len(packet)-1])
		values := strings.Split(packetStr, " ")
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

func HandlePutObjectRequest(
	ctx context.Context, oid, size string, user *user_model.User, repository *repo_model.Repository, a *PktAdapter,
) (int, error) {
	p := lfs_module.Pointer{Oid: oid}
	var err error
	if p.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return http.StatusUnprocessableEntity, fmt.Errorf("Invalid size: %q", size)
	}
	if !p.IsValid() {
		return http.StatusUnprocessableEntity, fmt.Errorf("Attempt to access invalid LFS OID[%s] in %s", p.Oid, repository.Name)
	}

	dataLen, dataReader, err := a.GetNextPacket()
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Cannot get next binary packet: %v", err)
	}
	if dataReader == nil {
		return http.StatusUnprocessableEntity, errors.New("Empty binary data")
	}
	if dataLen != p.Size {
		return http.StatusInternalServerError, fmt.Errorf("Data size mistmach: expecetd %d, got %d", p.Size, dataLen)
	}

	contentStore := lfs_module.NewContentStore()
	exists, err := contentStore.Exists(p)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Unable to check if LFS OID[%s] exist. Error: %v", p.Oid, err)
	}

	if exists {
		ok, err := quota_model.EvaluateForUser(ctx, user.ID, quota_model.LimitSubjectSizeGitLFS)
		if err != nil {
			return http.StatusInternalServerError, fmt.Errorf("quota_model.EvaluateForUser: %v", err)
		}
		if !ok {
			return http.StatusRequestEntityTooLarge, errors.New("quota exceeded")
		}
	}

	uploadOrVerify := func() error {
		if exists {
			accessible, err := git_model.LFSObjectAccessible(ctx, user, p.Oid)
			if err != nil {
				return fmt.Errorf("Unable to check if LFS MetaObject [%s] is accessible. Error: %v", p.Oid, err)
			}
			if !accessible {
				// The file exists but the user has no access to it.
				// The upload gets verified by hashing and size comparison to prove access to it.
				hash := sha256.New()
				written, err := io.Copy(hash, dataReader)
				if err != nil {
					return fmt.Errorf("Error creating hash. Error: %v", err)
				}

				if written != p.Size {
					return fmt.Errorf("content size does not match (got %d, expected %d)", written, p.Size)
					// return lfs_module.ErrSizeMismatch
				}
				if hex.EncodeToString(hash.Sum(nil)) != p.Oid {
					return lfs_module.ErrHashMismatch
				}
			} else {
				_, err := io.Copy(io.Discard, dataReader) // Discarding data
				if err != nil {
					return fmt.Errorf("Error discarding data: %v", err)
				}
			}
		} else if err := contentStore.Put(p, dataReader); err != nil {
			return err
		}
		dataLen, dataReader, err = a.GetNextPacket()
		if err != nil {
			return fmt.Errorf("Cannot get closing flush packet: %v", err)
		}
		if dataLen != 0 {
			return errors.New("Unexpected packet after binary data (should be flush)")
		}
		_, err := git_model.NewLFSMetaObject(ctx, repository.ID, p)
		return err
	}

	if err := uploadOrVerify(); err != nil {
		returnErr := fmt.Errorf("Error whilst uploadOrVerify LFS OID[%s]: %v", p.Oid, err)
		if _, err = git_model.RemoveLFSMetaObjectByOid(ctx, repository.ID, p.Oid); err != nil {
			returnErr = errors.Join(fmt.Errorf("Error whilst removing MetaObject for LFS OID[%s]: %v", p.Oid, err))
		}
		return http.StatusInternalServerError, returnErr
	}
	return http.StatusOK, nil
}

func VerifyObject(ctx context.Context, oid, size string) (int, error) {
	p := lfs_module.Pointer{Oid: oid}
	var err error
	if p.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return http.StatusUnprocessableEntity, fmt.Errorf("Invalid size: %q", size)
	}
	contentStore := lfs_module.NewContentStore()
	ok, err := contentStore.Verify(p)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error whilst verifying LFS OID[%s]: %v", p.Oid, err)
	} else if !ok {
		return http.StatusNotFound, fmt.Errorf("LFS OID[%s] not found", p.Oid)
	}
	return http.StatusOK, nil
}

func HandleGetObjectRequest(ctx context.Context, oid string, repository *repo_model.Repository, a *PktAdapter) (int, error) {
	p := lfs_module.Pointer{Oid: oid}
	if !p.IsValid() {
		return http.StatusUnprocessableEntity, errors.New("Oid or size are invalid")
	}
	meta, err := git_model.GetLFSMetaObjectByOid(ctx, repository.ID, p.Oid)
	if err != nil {
		return http.StatusNotFound, errors.New("Unable to get LFS oid")
	}

	contentStore := lfs_module.NewContentStore()
	content, err := contentStore.Get(meta.Pointer)
	if err != nil {
		return http.StatusNotFound, errors.New("Content not found")
	}
	defer content.Close()
	err = a.WriteBinaryData(content)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("Error whilst sendint LFS OID[%s] data: %v", p.Oid, err)
	}

	return http.StatusOK, nil
}
