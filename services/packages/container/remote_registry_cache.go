// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"fmt"
	"time"

	packages_model "forgejo.org/models/packages"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	packages_module "forgejo.org/modules/packages"
	container_module "forgejo.org/modules/packages/container"
	rr_module "forgejo.org/modules/packages/remote_registry"
	"forgejo.org/services/context"
	packages_service "forgejo.org/services/packages"
)

// SaveBlobToPackage saves a blob as a package
func SaveBlobToPackage(ctx *context.Context, buf *packages_module.HashedBuffer, remoteCtx *rr_module.RemoteRegistryContext, digest string, owner, creator *user_model.User) error {
	log.Debug("Saving blob %s as package", digest)
	pci := &packages_service.PackageCreationInfo{
		PackageInfo: packages_service.PackageInfo{
			Owner:   owner,
			Name:    remoteCtx.ImageName,
			Version: digest,
		},
		Creator: creator,
	}

	pb, pf, err := SaveAsPackageBlob(ctx, buf, pci)
	if err != nil {
		return fmt.Errorf("failed to save blob from remote registry: %w", err)
	}

	addRemoteMetadataToBlob(ctx, pb, remoteCtx, pf)

	return nil
}

// addRemoteMetadataToBlob Add rr id, time and remote digest as info to blob
func addRemoteMetadataToBlob(ctx *context.Context, pb *packages_model.PackageBlob, remoteCtx *rr_module.RemoteRegistryContext, pf *packages_model.PackageFile) {
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
