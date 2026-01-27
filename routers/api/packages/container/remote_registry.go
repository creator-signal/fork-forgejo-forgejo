// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"strings"

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

	imageStr := ctx.Params("image")
	isRemote := isRemoteRequest(imageStr)

	if !isRemote {
		// Not a remote registry request, continue with normal processing
		return
	}

	username := ctx.ContextUser.Name
	remoteName, imageName := parseImageName(imageStr)

	log.Trace("Detected remote registry request: owner=%s, remote=%s, image=%s",
		username, remoteName, imageName)

}

// Parse the request string for "remote" after {username}
func isRemoteRequest(imageStr string) bool {
	splitRequest := strings.Split(imageStr, "/")
	// this is the image namespace
	if splitRequest[0] == "remote" {
		return true
	}
	return false
}

// Parse the image string for its remote name and image name
func parseImageName(imageStr string) (string, string) {
	pathParts := strings.Split(imageStr, "/")
	return pathParts[1], pathParts[2]
}
