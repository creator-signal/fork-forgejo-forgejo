// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"fmt"
	"net/http"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

func getCachedRemoteManifest(ctx *context.Context) (*packages_model.PackageFileDescriptor, error) {
	// TODO Later we need a distinction between manifests from remote and local
	return getManifestFromContext(ctx)
}

// getLocalBlob finds a local blob if it exists, returns ErrContainerBlobNotExist otherwise
func getLocalBlob(ctx *context.Context, remoteCtx *RemoteRegistryContext, digest string) (*packages_model.PackageFileDescriptor, error) {
	opts := &container_model.BlobSearchOptions{
		OwnerID: ctx.Package.Owner.ID,
		Image:   remoteCtx.ImageName,
		Digest:  digest,
	}

	// Search for cached blob
	cached, err := workaroundGetContainerBlob(ctx, opts)
	if err != nil {
		if err == container_model.ErrContainerBlobNotExist {
			return nil, err
		}
		log.Error("Failed to check blob cache for %s: %v", digest, err)
		return nil, err
	}

	// Does the file exist?
	if cached.File == nil {
		return nil, container_model.ErrContainerBlobNotExist
	}

	// Get version properties to check PropertyRemoteSource
	versionProperties, err := packages_model.GetProperties(ctx, packages_model.PropertyTypeVersion, cached.File.VersionID)
	if err != nil {
		log.Warn("Failed to get version properties for cached blob: %v", err)
		return nil, container_model.ErrContainerBlobNotExist
	}

	var remoteSource string
	for _, prop := range versionProperties {
		if prop.Name == container_module.PropertyRemoteSource {
			remoteSource = prop.Value
			break
		}
	}

	expectedRemoteSource := fmt.Sprintf("%d", remoteCtx.RemoteRegistry.ID)

	if remoteSource != expectedRemoteSource {
		// Not from the same remote registry, treat as not cached
		return nil, container_model.ErrContainerBlobNotExist
	}

	return cached, nil
}

// saveBlobToPackage creates a package and saves it to the filesystem and db
func saveBlobToPackage(ctx *context.Context, buf *packages_module.HashedBuffer, imageName string) error {
	if _, err := saveAsPackageBlob(ctx,
		buf,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner: ctx.Package.Owner,
				Name:  imageName,
			},
			Creator: ctx.Doer,
		},
	); err != nil {
		switch err {
		case packages_service.ErrQuotaTotalCount, packages_service.ErrQuotaTypeSize, packages_service.ErrQuotaTotalSize:
			apiError(ctx, http.StatusForbidden, err)
		default:
			apiError(ctx, http.StatusInternalServerError, err)
		}
		return err
	}
	return nil
}
