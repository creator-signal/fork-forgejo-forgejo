// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/http"
	"strings"

	"forgejo.org/modules/log"
	container_module "forgejo.org/modules/packages/container"
	"forgejo.org/services/context"
	container_service "forgejo.org/services/packages/container"
)

func GetRemoteTagList(ctx *context.Context) {
}

func RemoteHeadBlob(ctx *context.Context) {
}

func RemoteGetBlob(ctx *context.Context) {
}

func RemoteHeadManifest(ctx *context.Context) {
	remoteCtx, err := GetRemoteRegistryContext(ctx)
	if err != nil {
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	reference := ctx.Params("reference")
	if reference == "" {
		apiErrorDefined(ctx, errManifestUnknown)
		return
	}

	// Do we have the manifest cached locally?
	manifest, err := getCachedRemoteManifest(ctx)
	if manifest != nil && err == nil {
		setResponseHeaders(ctx.Resp, &containerHeaders{
			ContentDigest: manifest.Properties.GetByName(container_module.PropertyDigest),
			ContentType:   manifest.Properties.GetByName(container_module.PropertyMediaType),
			ContentLength: manifest.Blob.Size,
			Status:        http.StatusOK,
		})
		log.Trace("Remote manifest with file ID: %s existed", manifest.File.ID)
		return
	}

	// Not cached, fetch from remote registry
	client, err := container_service.NewContainerRegistryClient(remoteCtx.RemoteRegistry)
	if err != nil {
		log.Error("Failed to create remote registry client for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Manifest Ref
	r, err := client.NewRef(remoteCtx.ImageName)
	if err != nil {
		log.Error("Failed to create reference for %s: %v", remoteCtx.RemoteRegistry.Name, err)
		apiError(ctx, http.StatusInternalServerError, err)
		return
	}
	defer client.Close(ctx, r)

	// Get manifest metadata from remote registry
	manifestResp, err := client.HeadManifest(ctx, r)
	if err != nil {
		log.Error("Failed to HEAD manifest %s:%s from remote registry %s: %v",
			remoteCtx.ImageName, reference, remoteCtx.RemoteRegistry.Name, err)

		if strings.Contains(err.Error(), "404") {
			apiErrorDefined(ctx, errManifestUnknown)
		} else if strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "403") {
			apiErrorDefined(ctx, errUnauthorized)
		} else {
			apiError(ctx, http.StatusBadGateway, err)
		}
		return
	}

	// Set response headers from remote
	setResponseHeaders(ctx.Resp, &containerHeaders{
		ContentDigest: manifestResp.GetRef().Digest,
		ContentType:   manifestResp.GetDescriptor().MediaType,
		ContentLength: manifestResp.GetDescriptor().Size,
		Status:        http.StatusOK,
	})
}

func RemoteGetManifest(ctx *context.Context) {
}
