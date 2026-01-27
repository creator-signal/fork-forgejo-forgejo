// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const remoteRegistryContextKey = "RemoteRegistryContext"

// Unsupported operations for remote registries
func remoteUnsupportedOperation(ctx *context.Context) {
	apiErrorDefined(ctx, errUnsupported.WithMessage("Pushing to remote registries is not supported"))
}

// HandleGetBlob handles blob requests
func HandleGetBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteGetBlob, GetBlob)
}

// HandleHeadBlob handles blob head requests
func HandleHeadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteHeadBlob, HeadBlob)
}

// HandleGetManifest handles manifest requests
func HandleGetManifest(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteGetManifest, GetManifest)
}

// HandleHeadManifest handles manifest head requests
func HandleHeadManifest(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteHeadManifest, HeadManifest)
}

// HandleInitiateUploadBlob handles upload initiation (remote uploads not supported)
func HandleInitiateUploadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, InitiateUploadBlob)
}

// HandleUploadManifest handles manifest uploads (remote uploads not supported)
func HandleUploadManifest(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, UploadManifest)
}

// HandleDeleteBlob handles blob deletion (remote deletes not supported)
func HandleDeleteBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, DeleteBlob)
}

// HandleDeleteManifest handles manifest deletion (remote deletes not supported)
func HandleDeleteManifest(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, DeleteManifest)
}

// Upload operations that need special handling (all unsupported for remote)
func HandleGetUploadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, GetUploadBlob)
}

// HandleUploadBlob handles the uploading of blobs (unsupported for remote)
func HandleUploadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, UploadBlob)
}

// HandleEndUploadBlob handles stopping the uploading of blobs (unsupported for remote)
func HandleEndUploadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, EndUploadBlob)
}

// HandleEndUploadBlob handles cancelation of uploading blobs (unsupported for remote)
func HandleCancelUploadBlob(ctx *context.Context) {
	HandleRegistryRequest(ctx, remoteUnsupportedOperation, CancelUploadBlob)
}

// HandleEndUploadBlob handles cancelation of uploading blobs (unsupported for remote)
func HandleVerifyImageName(ctx *context.Context) {
	HandleRegistryRequest(ctx, VerifyRemoteImageName, VerifyImageName)
}

func GetRemoteRegistryContext(ctx *context.Context) *RemoteRegistryContext {
	if remoteCtx, ok := ctx.Data[remoteRegistryContextKey].(*RemoteRegistryContext); ok {
		return remoteCtx
	}
	return nil
}

// IsRemoteRegistryRequest checks if the current request is for a remote registry
func IsRemoteRegistryRequest(ctx *context.Context) bool {
	remoteCtx := GetRemoteRegistryContext(ctx)
	return remoteCtx != nil && remoteCtx.IsRemoteRequest
}

func HandleRegistryRequest(ctx *context.Context, remoteFunc, localFunc func(ctx *context.Context)) {
	if IsRemoteRegistryRequest(ctx) && setting.Packages.RemoteRegistry.Enabled {
		remoteFunc(ctx)
	} else {
		localFunc(ctx)
	}
}
