// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/http"

	"forgejo.org/services/context"
)

func VerifyRemoteImageName(ctx *context.Context) {
	return
}

func GetRemoteTagList(ctx *context.Context) {
	return
}

func RemoteHeadBlob(ctx *context.Context) {
	return
}

func RemoteGetBlob(ctx *context.Context) {
	return
}

func RemoteHeadManifest(ctx *context.Context) {
	setResponseHeaders(ctx.Resp, &containerHeaders{
		Status: http.StatusOK,
	})
}

func RemoteGetManifest(ctx *context.Context) {
	return
}
