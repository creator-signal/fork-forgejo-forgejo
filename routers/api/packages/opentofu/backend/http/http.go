package http

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	packages_model "forgejo.org/models/packages"
	opentofu_model "forgejo.org/models/packages/opentofu"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	packages_module "forgejo.org/modules/packages"
	opentofu_lock_module "forgejo.org/modules/packages/opentofu/lock"
	opentofu_state_module "forgejo.org/modules/packages/opentofu/state"
	"forgejo.org/modules/setting"
	"forgejo.org/routers/api/packages/helper"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

// apiError logs and processes a REST API error.
func apiError(ctx *context.Context, status int, obj any) {
	type Error struct {
		Code    string `json:"code"`
		Message string `json:"message,omitempty"`
	}

	helper.LogAndProcessError(ctx, status, obj, func(message string) {
		ctx.JSON(status, Error{
			Code:    http.StatusText(status),
			Message: message,
		})
	})
}

// GetState returns the latest version of the state file if any.
func GetState(ctx *context.Context) {
	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package fetch request: %s [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Search for the latest versions of the package/state file.
	pvs, _, err := packages_model.SearchLatestVersions(ctx, &packages_model.PackageSearchOptions{
		OwnerID:         ctx.Package.Owner.ID,
		Type:            packages_model.TypeOpenTofuState,
		Name:            packages_model.SearchValue{ExactMatch: true, Value: packageName},
		HasFileWithName: opentofu_state_module.OpenTofuStateFilename,
		IsInternal:      optional.Some(false),
		Sort:            packages_model.SortCreatedDesc,
	})
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to search for the latest versions of the state file: %w", err))
		return
	}

	// If the package/state file does not exist.
	if len(pvs) == 0 {
		log.Debug("No state file available to download for package %s", packageName)
		ctx.Status(http.StatusNoContent)
		return
	}

	// Get the latest version of the package/state file now that we know its version
	// number.
	s, _, pf, err := packages_service.GetFileStreamByPackageNameAndVersion(
		ctx,
		&packages_service.PackageInfo{
			Owner:       ctx.Package.Owner,
			PackageType: packages_model.TypeOpenTofuState,
			Name:        packageName,
			Version:     pvs[0].Version,
		},
		&packages_service.PackageFileInfo{
			Filename: opentofu_state_module.OpenTofuStateFilename,
		},
	)
	if err != nil {
		switch {
		case errors.Is(err, packages_model.ErrPackageNotExist):
			apiError(ctx, http.StatusNotFound, "the package cannot be found")
		case errors.Is(err, packages_model.ErrPackageFileNotExist):
			apiError(ctx, http.StatusNotFound, "the state file cannot be found in the package")
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to find the package and its state file: %w", err))
		}

		return
	}
	defer s.Close()

	log.Debug("Package %s successfully fetched [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Return the state file.
	helper.ServePackageFile(ctx, s, nil, pf, &context.ServeHeaderOptions{
		ContentType:  "application/json",
		Filename:     pf.Name,
		LastModified: pf.CreatedUnix.AsLocalTime(),
	})
}

// UpdateState processes the REST API requests received to create/update an
// OpenTofu/Terraform state file as Forgejo package.
func UpdateState(ctx *context.Context) {
	defer ctx.Req.Body.Close()

	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package update request: %s [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Get the state lock from the database.
	lock, err := opentofu_model.GetLock(ctx, packageName, ctx.Package.Owner.ID)
	if err != nil {
		if !opentofu_model.IsErrStateLockNotExist(err) {
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to get the state lock status: %w", err))
			return
		}
	}

	// If the state file is locked.
	if lock != nil {
		// Get the lock ID from the request.
		lockID := ctx.Req.URL.Query().Get("ID")

		// If the caller is not the one having locked the state file.
		if lockID == "" {
			apiError(ctx, http.StatusConflict, "the state file is locked, but no lock ID was provided")
			return
		} else if lockID != lock.LockID {
			apiError(ctx, http.StatusUnauthorized, fmt.Errorf("invalid lock ID: %s", lockID))
			return
		}
	}

	// Check the size of the state file.
	contentLength := ctx.Req.ContentLength
	log.Debug("Update request's content length: %d", contentLength)
	if contentLength == -1 {
		apiError(ctx, http.StatusLengthRequired, "the content length is unknown")
		return
	} else if contentLength == 0 {
		apiError(ctx, http.StatusBadRequest, "the body is empty")
		return
	} else if setting.Packages.LimitSizeOpenTofuState > -1 && contentLength > setting.Packages.LimitSizeOpenTofuState {
		apiError(ctx, http.StatusRequestEntityTooLarge, "request body exceeds the package size limit defined by the server")
		return
	}

	// Read the state file from the request body.
	//
	// The amount of bytes to read is limited by the value of the request's content
	// length to avoid denial of service attacks.
	stateFile, err := io.ReadAll(http.MaxBytesReader(ctx.Resp, ctx.Req.Body, contentLength))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to read the state file from the request body: %w", err))
		return
	}

	var md5Hash [16]byte

	// If the request contains an MD5 checksum in its headers, check if it matches
	// the request body.
	md5Checksum := ctx.Req.Header.Get("Content-MD5")
	if md5Checksum != "" {
		log.Debug("Update request has an MD5 checksum: %s", md5Checksum)

		md5Hash = md5.Sum(stateFile)
		md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

		if md5Checksum != md5Base64 {
			apiError(ctx, http.StatusBadRequest, "the MD5 checksum sent with the request does not match the body content")
			return
		}
	}

	// Parse the state file to extract metadata.
	metadata, err := opentofu_state_module.ParseMetadataFromStateFile(&stateFile, &md5Hash)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, fmt.Errorf("failed to parse the state file: %w", err))
		return
	}

	// Prepare the state file to be stored as a Forgejo package.
	packageData, err := packages_module.CreateHashedBufferFromReader(bytes.NewReader(stateFile))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to create an hashed buffer from the state file: %w", err))
		return
	}
	defer packageData.Close()

	// Create the package.
	_, _, err = packages_service.CreatePackageAndAddFile(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       ctx.Package.Owner,
				PackageType: packages_model.TypeOpenTofuState,
				Name:        packageName,
				Version:     strconv.FormatUint(metadata.Serial, 10),
			},
			Creator:  ctx.Doer,
			Metadata: metadata,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: opentofu_state_module.OpenTofuStateFilename,
			},
			Creator: ctx.Doer,
			Data:    packageData,
			IsLead:  true,
		},
	)
	if err != nil {
		switch err {
		case packages_model.ErrDuplicatePackageVersion:
			apiError(ctx, http.StatusConflict, "a package with the same version number already exists")
		case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
			apiError(ctx, http.StatusForbidden, fmt.Errorf("quota exceeded: %v", err))
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to create the package and add the state file to it: %w", err))
		}

		return
	}

	log.Debug("Package %s successfully uploaded [OwnerID: %d]", packageName, ctx.Package.Owner.ID)
	ctx.Status(http.StatusCreated)
}

// DeleteState deletes all versions of the package/state file.
func DeleteState(ctx *context.Context) {
	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package deletion request: %s [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Get all versions of the package/state file.
	pvs, err := packages_model.GetVersionsByPackageName(ctx, ctx.Package.Owner.ID, packages_model.TypeOpenTofuState, packageName)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to get all versions of the state file: %w", err))
		return
	}

	// If no package versions were found.
	if len(pvs) == 0 {
		apiError(ctx, http.StatusNotFound, "no package versions found")
		return
	}

	for _, pv := range pvs {
		if err := packages_service.RemovePackageVersion(ctx, ctx.Doer, pv); err != nil {
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to delete a state file version: %w", err))
			return
		}
	}

	log.Debug("Package %s successfully deleted [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// An HTTP status 204 No Content would be more appropriate here, but OpenTofu
	// and Terraform only accept HTTP 200 OK status.
	ctx.Status(http.StatusOK)
}

// LockState locks a Forgejo package/state file.
func LockState(ctx *context.Context) {
	defer ctx.Req.Body.Close()

	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package lock request: %s [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Check the size of the lock request body.
	contentLength := ctx.Req.ContentLength
	log.Debug("Lock request's content length: %d", contentLength)
	if contentLength == -1 {
		apiError(ctx, http.StatusLengthRequired, "the content length is unknown")
		return
	} else if contentLength == 0 {
		apiError(ctx, http.StatusBadRequest, "the body is empty")
		return
	} else if contentLength > opentofu_lock_module.LimitSizeLockInfo {
		apiError(ctx, http.StatusRequestEntityTooLarge, "request body exceeds the request handler limit")
		return
	}

	// Read the state lock payload from the request body.
	//
	// The amount of bytes to read is limited by the value of the request's content
	// length to avoid denial of service attacks.
	stateLockPayload, err := io.ReadAll(http.MaxBytesReader(ctx.Resp, ctx.Req.Body, contentLength))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to read the state lock payload from the request body: %w", err))
		return
	}

	// Parse the lock request payload to extract the lock information.
	lockInfo, err := opentofu_lock_module.ParseLockInfo(&stateLockPayload)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, fmt.Errorf("failed to parse the lock request payload: %w", err))
		return
	}

	// Add the missing fields from the lock request payload.
	lockInfo.PackageName = packageName
	lockInfo.OwnerID = ctx.Package.Owner.ID

	// Replace the username sent by the client with the caller's Forgejo username.
	lockInfo.UserName = ctx.Doer.Name

	// Lock the state file.
	stateLock, err := opentofu_model.Lock(ctx, lockInfo)
	if err != nil {
		switch {
		case opentofu_model.IsErrStateLockAlreadyExist(err):
			log.Debug("The state file is already locked: %w", err)
			ctx.JSON(http.StatusConflict, stateLock)
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to lock the state file: %w", err))
		}

		return
	}

	log.Debug("Package %s is now locked [OwnerID: %d]", packageName, ctx.Package.Owner.ID)
	ctx.JSON(http.StatusOK, stateLock)
}

// UnlockState unlocks a Forgejo package/state file.
func UnlockState(ctx *context.Context) {
	defer ctx.Req.Body.Close()

	// Get the package name from the request.
	packageName := ctx.Params("packagename")
	log.Debug("Processing OpenTofu/Terraform HTTP backend package unlock request: %s [OwnerID: %d]", packageName, ctx.Package.Owner.ID)

	// Check the size of the unlock request body.
	contentLength := ctx.Req.ContentLength
	log.Debug("Unlock request's content length: %d", contentLength)
	if contentLength == -1 {
		apiError(ctx, http.StatusLengthRequired, "the content length is unknown")
		return
	} else if contentLength == 0 {
		apiError(ctx, http.StatusBadRequest, "the body is empty")
		return
	} else if contentLength > opentofu_lock_module.LimitSizeLockInfo {
		apiError(ctx, http.StatusRequestEntityTooLarge, "request body exceeds the request handler limit")
		return
	}

	// Read the state unlock payload from the request body.
	//
	// The amount of bytes to read is limited by the value of the request's content
	// length to avoid denial of service attacks.
	stateUnlockPayload, err := io.ReadAll(http.MaxBytesReader(ctx.Resp, ctx.Req.Body, contentLength))
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to read the state unlock payload from the request body: %w", err))
		return
	}

	// Parse the unlock request payload to extract the lock information.
	lockInfo, err := opentofu_lock_module.ParseLockInfo(&stateUnlockPayload)
	if err != nil {
		apiError(ctx, http.StatusBadRequest, fmt.Errorf("failed to parse the unlock request payload: %w", err))
		return
	}

	// Unlock the state file.
	err = opentofu_model.Unlock(ctx, packageName, ctx.Package.Owner.ID, lockInfo.LockID)
	if err != nil {
		switch {
		case opentofu_model.IsErrStateLockNotExist(err):
			apiError(ctx, http.StatusNotFound, fmt.Errorf("the state file is not locked: %w", err))
		case opentofu_model.IsErrInvalidLockID(err):
			apiError(ctx, http.StatusUnauthorized, fmt.Errorf("wrong lock ID: %w", err))
		default:
			apiError(ctx, http.StatusInternalServerError, fmt.Errorf("failed to unlock the state file: %w", err))
		}

		return
	}

	log.Debug("Package %s is now unlocked [OwnerID: %d]", packageName, ctx.Package.Owner.ID)
	ctx.Status(http.StatusOK)
}
