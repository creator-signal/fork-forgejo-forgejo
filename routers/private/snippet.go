// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package private

import (
	"fmt"
	"net/http"

	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/modules/log"
	"forgejo.org/modules/private"
	app_context "forgejo.org/services/context"
	snippet_service "forgejo.org/services/snippet"
)

func SnippetUpdated(ctx *app_context.PrivateContext) {
	snippetUUID := ctx.Params(":snippet_uuid")

	snippet, err := snippet_model.GetSnippetByUUID(ctx, snippetUUID)
	if err != nil {
		log.Error("Failed to get snippet: %s Error: %v", snippetUUID, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Failed to get snippet: %s Error: %v", snippetUUID, err),
		})
		return
	}

	files, err := snippet_service.GetFiles(ctx, snippet)
	if err != nil {
		log.Error("Failed to get snippet files: %s Error: %v", snippet.UUID, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Failed to get snippet files: %s Error: %v", snippet.UUID, err),
		})
		return
	}

	snippet.Language = files.GetLanguage()
	err = snippet.UpdateCols(ctx, "language")
	if err != nil {
		log.Error("Failed to get update snippet: %s Error: %v", snippet.UUID, err)
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Failed to get update snippet: %s Error: %v", snippet.UUID, err),
		})
		return
	}
}
