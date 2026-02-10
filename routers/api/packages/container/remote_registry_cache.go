// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
	container_service "forgejo.org/services/packages/container"
	"github.com/regclient/regclient/types/manifest"
)

func getCachedRemoteManifest(ctx *context.Context) (*packages_model.PackageFileDescriptor, error) {
	// TODO Later we need a distinction between manifests from remote and local
	return getManifestFromContext(ctx)
}

// getLocalBlob finds a local blob if it exists, returns ErrContainerBlobNotExist otherwise
func getLocalBlob(ctx *context.Context, remoteCtx *RemoteRegistryContext, digest string) (*packages_model.PackageFileDescriptor, error) {
	opts := &container_model.BlobSearchOptions{
		OwnerID: ctx.ContextUser.ID,
		Image:   remoteCtx.ImageName,
		Digest:  digest,
	}

	// Get blob or err
	log.Debug("Trying to find blob %s locally", digest)
	blobDescriptor, err := workaroundGetContainerBlob(ctx, opts)
	if err != nil {
		if err == container_model.ErrContainerBlobNotExist {
			return nil, err
		}
		log.Error("Failed to check blob cache for %s: %v", digest, err)
		return nil, err
	}

	// Update cache time (if it exists), as we are using this blob again
	pf, err := packages_model.GetFileForVersionByName(
		ctx,
		blobDescriptor.File.VersionID,
		blobDescriptor.File.LowerName,
		packages_model.EmptyFileKey)
	if err != nil {
		log.Error("Could not find file for blob %s: %v", digest, err)
		return nil, err
	}
	err = packages_model.UpdateProperty(ctx,
		&packages_model.PackageProperty{
			RefType: packages_model.PropertyTypeFile,
			RefID:   pf.ID,
			Name:    container_module.PropertyCacheTime,
			Value:   fmt.Sprintf("%d", time.Now().Unix()),
		})
	if err != nil {
		log.Warn("Failed to set/update blob property %s for remote blob: %v", container_module.PropertyCacheTime, err)
	}

	return blobDescriptor, nil
}

// saveBlobToPackage saves a blob as a package
func saveBlobToPackage(ctx *context.Context, buf *packages_module.HashedBuffer, remoteCtx *RemoteRegistryContext, digest string, owner, creator *user_model.User) error {
	log.Debug("Saving blob %s as package", digest)
	pci := &packages_service.PackageCreationInfo{
		PackageInfo: packages_service.PackageInfo{
			Owner:   owner,
			Name:    remoteCtx.ImageName,
			Version: digest,
		},
		Creator: creator,
	}

	pb, pf, err := saveAsPackageBlob(ctx, buf, pci)
	if err != nil {
		return fmt.Errorf("failed to save blob from remote registry: %w", err)
	}

	addRemoteMetadataToBlob(ctx, pb, remoteCtx, pf)

	return nil
}

// addRemoteMetadataToBlob Add rr id, time and remote digest as info to blob
func addRemoteMetadataToBlob(ctx *context.Context, pb *packages_model.PackageBlob, remoteCtx *RemoteRegistryContext, pf *packages_model.PackageFile) {
	// Add remote registry metadata
	properties := map[string]string{
		container_module.PropertyRemoteSource: fmt.Sprintf("%d", remoteCtx.RemoteRegistry.ID),
		container_module.PropertyCacheTime:    fmt.Sprintf("%d", time.Now().Unix()),
		container_module.PropertyRemoteDigest: fmt.Sprintf("sha256:%s", pb.HashSHA256),
	}

	for name, value := range properties {
		if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypeFile, pf.ID, name, value); err != nil {
			log.Warn("Failed to set blob property %s for remote blob: %v", name, err)
		}
	}
}

func saveManifest(ctx *context.Context, man manifest.Manifest) error {
	mci, err := container_service.NewRemoteManifestCreationInfo(
		ctx.ContextUser,
		ctx.Doer,
		man.GetDescriptor().MediaType,
		ctx.Params("image"),
		ctx.Params("reference"), // TODO Reference may need to be a tag as well
		man.GetRef().Registry,
	)
	if err != nil {
		apiErrorDefined(ctx, errManifestInvalid.WithMessage(err.Error()))
		return err
	}

	maxSize := maxManifestSize + 1
	b, err := man.RawBody()
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return err
	}

	reader := bytes.NewReader(b)
	lReader := &io.LimitedReader{R: reader, N: int64(maxSize)}
	buf, err := packages_module.CreateHashedBufferFromReaderWithSize(lReader, maxSize)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return err
	}
	defer buf.Close()

	if buf.Size() > maxManifestSize {
		apiErrorDefined(ctx, errManifestInvalid.WithMessage("Manifest exceeds maximum size").WithStatusCode(http.StatusRequestEntityTooLarge))
		return err
	}

	_, err = processManifest(ctx, mci, buf)
	if err != nil {
		var namedError *namedError
		if errors.As(err, &namedError) {
			apiErrorDefined(ctx, namedError)
		} else if errors.Is(err, container_model.ErrContainerBlobNotExist) {
			apiErrorDefined(ctx, errBlobUnknown)
		} else {
			switch err {
			case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
				apiError(ctx, http.StatusForbidden, err)
			default:
				apiError(ctx, http.StatusInternalServerError, err)
			}
		}
		return err
	}
	return nil
}
