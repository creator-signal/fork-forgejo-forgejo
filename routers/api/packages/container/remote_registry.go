// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/http"

	"forgejo.org/services/context"

	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	rr_service "forgejo.org/services/packages"
)

// RemoteRegistryContext represents remote registry information in the request context
type RemoteRegistryContext struct {
	RemoteRegistry *rr_model.RemoteRegistry
	OwnerName      string
	ImageName      string
}

const remoteRegistryContextKey = "RemoteRegistryContext"

// RemoteRegistryMiddleware detects and resolves remote registry requests
func RemoteRegistryMiddleware(ctx *context.Context) {
	if !setting.Packages.RemoteRegistry.Enabled {
		return
	}

	registryName := ctx.Params("registry-name")
	ownerName := ctx.Params("username")
	imageName := ctx.Params("image")
	username := ctx.ContextUser.Name
	isOrg := ctx.ContextUser.IsOrganization()
	isUser := ctx.ContextUser.IsUser()

	log.Trace("Detected remote registry request: owner=%s, user=%s, remote=%s, image=%s",
		ownerName, username, registryName, imageName)

	// Resolve remote registry with precedence
	remoteRegistry, err := rr_service.GetRemoteRegistry(ctx, isOrg, isUser, ownerName, registryName)
	if err != nil {
		log.Error("Failed to resolve remote registry %q: %v", registryName, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	remoteCtx := &RemoteRegistryContext{
		OwnerName:      ownerName,
		ImageName:      imageName,
		RemoteRegistry: remoteRegistry,
	}

	// Store in context
	ctx.Data[remoteRegistryContextKey] = remoteCtx
}
