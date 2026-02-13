// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"errors"
	"net/http"

	"forgejo.org/modules/log"
	rr_module "forgejo.org/modules/packages/remote_registry"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	rr_service "forgejo.org/services/packages"
)

const remoteRegistryContextKey = "RemoteRegistryContext"

// RemoteRegistryMiddleware detects and resolves remote registry requests
func RemoteRegistryMiddleware(ctx *context.Context) {
	if !setting.Packages.RemoteRegistry.Enabled {
		return
	}

	registryName := ctx.Params("registry-name")
	ownerName := ctx.Params("username")
	imageName := ctx.Params("image")
	reference := ctx.Params("reference")
	dig := ctx.Params("diges")
	username := ctx.ContextUser.Name
	isOrg := ctx.ContextUser.IsOrganization()
	isUser := ctx.ContextUser.IsUser()

	log.Trace("Detected remote registry request: owner=%s, user=%s, remote=%s, image=%s",
		ownerName, username, registryName, imageName)

	remoteRegistry, err := rr_service.GetRemoteRegistry(ctx, isOrg, isUser, ownerName, registryName)
	if err != nil {
		log.Error("Failed to resolve remote registry %q: %v", registryName, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	remoteCtx := &rr_module.RemoteRegistryContext{
		OwnerName:      ownerName,
		ImageName:      imageName,
		RemoteRegistry: remoteRegistry,
		Reference:      reference,
		Digest:         dig,
	}

	ctx.Data[remoteRegistryContextKey] = remoteCtx
}

func GetRemoteRegistryContext(ctx *context.Context) (*rr_module.RemoteRegistryContext, error) {
	remoteCtx, ok := ctx.Data[remoteRegistryContextKey].(*rr_module.RemoteRegistryContext)
	if !ok {
		return &rr_module.RemoteRegistryContext{}, errors.New("Remote registry context not found")
	}
	return remoteCtx, nil
}
