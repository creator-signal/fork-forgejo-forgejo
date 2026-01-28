// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"forgejo.org/services/context"

	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

// RemoteRegistryContext represents remote registry information in the request context
type RemoteRegistryContext struct {
	IsRemoteRequest   bool
	RemoteRegistry    *rr_model.RemoteRegistry
	OwnerName         string
	RemoteName        string
	ImageName         string
	OriginalImageName string // the parsed image part before remote resolution
}

// RemoteRegistryMiddleware detects and resolves remote registry requests
func RemoteRegistryMiddleware(ctx *context.Context) {
	if !setting.Packages.RemoteRegistry.Enabled {
		return
	}

	remoteName := ctx.Params("remote-name")
	ownerName := ctx.Params("username")
	imageName := ctx.Params("image")
	username := ctx.ContextUser.Name

	log.Trace("Detected remote registry request: owner=%s, user=%s, remote=%s, image=%s",
		ownerName, username, remoteName, imageName)

}
