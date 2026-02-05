// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	go_context "context"
	"fmt"
	"strings"
	"time"

	packages_model "forgejo.org/models/packages"
	container_model "forgejo.org/models/packages/container"
	user_model "forgejo.org/models/user"
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
func saveBlobToPackage(ctx go_context.Context, buf *packages_module.HashedBuffer, remoteCtx *RemoteRegistryContext, digest string, owner, creator *user_model.User) error {
	log.Debug("Saving blob %s as package", digest)
	pci := &packages_service.PackageCreationInfo{
		PackageInfo: packages_service.PackageInfo{
			Owner:   owner,
			Name:    remoteCtx.ImageName,
			Version: digest,
		},
		Creator: creator,
	}

	pb, err := saveAsPackageBlob(ctx, buf, pci)
	if err != nil {
		return fmt.Errorf("failed to save blob from remote registry: %w", err)
	}

	pv, err := packages_model.GetVersionByNameAndVersion(ctx, owner.ID, packages_model.TypeContainer, pci.PackageInfo.Name, pci.PackageInfo.Version)
	if err != nil {
		return fmt.Errorf("package version not found: %w", err)
	}

	err = addRemoteMetadataToBlob(ctx, pb, pv.ID, remoteCtx)
	if err != nil {
		return fmt.Errorf("failed to add metadata to blob: %w", err)
	}

	return nil
}

// addRemoteMetadataToBlob adds tracking properties to cached blob
func addRemoteMetadataToBlob(ctx go_context.Context, pb *packages_model.PackageBlob, versionId int64, remoteCtx *RemoteRegistryContext) error {
	// Find the package file for this blob
	filename := strings.ToLower(fmt.Sprintf("sha256_%s", pb.HashSHA256))
	pf, err := packages_model.GetFileForVersionByName(ctx, versionId, filename, packages_model.EmptyFileKey)
	if err != nil {
		return fmt.Errorf("failed to find package file for cached blob: %w", err)
	}

	// Add remote registry metadata
	properties := map[string]string{
		container_module.PropertyRemoteSource: fmt.Sprintf("%d", remoteCtx.RemoteRegistry.ID),
		container_module.PropertyCacheTime:    fmt.Sprintf("%d", time.Now().Unix()),
		container_module.PropertyRemoteDigest: fmt.Sprintf("sha256:%s", pb.HashSHA256),
	}

	for name, value := range properties {
		if _, err := packages_model.InsertProperty(ctx, packages_model.PropertyTypeFile, pf.ID, name, value); err != nil {
			log.Warn("Failed to set blob property %s for remote blob: %v", name, err)
			// Continue even if property setting fails
		}
	}

	return nil
}
