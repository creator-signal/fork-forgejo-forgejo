// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"fmt"

	snippet_model "forgejo.org/models/snippet"
)

// SnippetAssignment handles Context.Snippet assignment
func SnippetAssignment(ctx *Context) {
	snippetUUID := ctx.Params(":snippet_uuid")

	snippet, err := snippet_model.GetSnippetByUUID(ctx, snippetUUID)
	if err != nil {
		if snippet_model.IsErrSnippetNotExist(err) {
			ctx.NotFound(fmt.Sprintf("snippet %s was not found", snippetUUID), nil)
		} else {
			ctx.ServerError("GetSnippetByUUID", err)
		}
		return
	}

	if !snippet.HasAccess(ctx.Doer) {
		ctx.NotFound(fmt.Sprintf("snippet %s is private", snippetUUID), nil)
		return
	}

	ctx.Snippet = snippet
}

// RequireSnippetOwner checks if teh Doer is the Owner of the Snippet
func RequireSnippetOwner(ctx *Context) {
	if !ctx.Snippet.IsOwner(ctx.Doer) {
		ctx.NotFound(ctx.Req.URL.RequestURI(), nil)
	}
}
