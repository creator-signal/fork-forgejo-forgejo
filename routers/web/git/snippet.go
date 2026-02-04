// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"net/http"
	"strings"

	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/services/context"
)

type serviceHandlerSnippet struct {
	snippet *snippet_model.Snippet
}

func (h *serviceHandlerSnippet) Init(ctx *context.Context, isPull, receivePack bool) bool {
	snippetUUID := strings.TrimSuffix(strings.ToLower(ctx.Params(":snippet_uuid")), ".git")

	var err error

	h.snippet, err = snippet_model.GetSnippetByUUID(ctx, snippetUUID)
	if err != nil {
		if snippet_model.IsErrSnippetNotExist(err) {
			if !ctx.IsSigned {
				ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
				ctx.Error(http.StatusUnauthorized)
				return false
			}

			ctx.NotFound("", nil)
		} else {
			ctx.ServerError("GetSnippetByUUID", err)
		}
		return false
	}

	if !h.snippet.HasAccess(ctx.Doer) {
		if !ctx.IsSigned {
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
			ctx.Error(http.StatusUnauthorized)
			return false
		}

		ctx.NotFound("", nil)
		return false
	}

	if !isPull {
		if !ctx.IsSigned {
			ctx.Resp.Header().Set("WWW-Authenticate", `Basic realm="Gitea"`)
			ctx.Error(http.StatusUnauthorized)
			return false
		}

		if !h.snippet.IsOwner(ctx.Doer) {
			ctx.PlainText(http.StatusForbidden, "not the owner")
		}
	}

	return true
}

func (h *serviceHandlerSnippet) GetRepoPath() string {
	return h.snippet.GetRepoPath()
}

func (h *serviceHandlerSnippet) GetEnviron() []string {
	return make([]string, 0)
}
