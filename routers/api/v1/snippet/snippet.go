// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"net/http"

	snippet_model "forgejo.org/models/snippet"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/web"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
	snippet_service "forgejo.org/services/snippet"
)

// Search for Snippets
func Search(ctx *context.APIContext) {
	// swagger:operation GET /snippets/search snippet searchSnippets
	// ---
	// summary: Search for snippets
	// produces:
	// - application/json
	// parameters:
	// - name: q
	//   in: query
	//   description: keyword
	//   type: string
	// - name: owner_id
	//   in: query
	//   description: search only for repos that the user with the given id owns
	//   type: integer
	//   format: int64
	// - name: sort
	//   in: query
	//   description: sort snippets by attribute
	//   type: string
	//   enum: [newest, oldest, alphabetically, reversealphabetically]
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/SnippetList"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	opts := &snippet_model.SearchSnippetsOptions{
		ListOptions: utils.GetListOptions(ctx),
		Actor:       ctx.Doer,
		Keyword:     ctx.FormTrim("q"),
		OwnerID:     ctx.FormInt64("uid"),
		SortOrder:   ctx.FormTrim("sort"),
	}

	snippets, count, err := snippet_model.SearchSnippets(ctx, opts)
	if err != nil {
		ctx.ServerError("SearchSnippets", err)
		return
	}

	err = snippets.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	ctx.SetLinkHeader(int(count), opts.PageSize)
	ctx.SetTotalCountHeader(count)
	ctx.JSON(http.StatusOK, convert.ToSnippetList(ctx, snippets, ctx.Doer))
}

// Create a snippet
func Create(ctx *context.APIContext) {
	// swagger:operation POST /snippets snippet createSnippet
	// ---
	// summary: Create a Snippet
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/CreateSnippetOption"
	// responses:
	//   "201":
	//     "$ref": "#/responses/Snippet"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "422":
	//     "$ref": "#/responses/validationError"
	opt := web.GetForm(ctx).(*api.CreateSnippetOption)

	if len(opt.Files) == 0 {
		ctx.Error(http.StatusBadRequest, "files can't be empty", nil)
		return
	}

	visibility, err := snippet_model.SnippetVisibilityFromName(opt.Visibility)
	if err != nil {
		ctx.Error(http.StatusBadRequest, "invalid visibility", nil)
		return
	}

	if len(opt.Files) == 0 {
		ctx.Error(http.StatusBadRequest, "no files", nil)
		return
	}

	files := snippet_service.SnippetFiles(opt.Files)
	err = files.Validate()
	if err != nil {
		ctx.Error(http.StatusBadRequest, err.Error(), nil)
		return
	}

	snippet, err := snippet_service.CreateSnippet(ctx, ctx.Doer, opt.Name, opt.Description, visibility, files)
	if err != nil {
		ctx.ServerError("CreateSnippet", err)
		return
	}

	ctx.JSON(http.StatusCreated, convert.ToSnippet(ctx, snippet, ctx.Doer))
}

// Get a snippet
func Get(ctx *context.APIContext) {
	// swagger:operation GET /snippets/{snippet_uuid} snippet getSnippet
	// ---
	// summary: Get a Snippet
	// produces:
	// - application/json
	// parameters:
	// - name: snippet_uuid
	//   in: path
	//   description: uuid of the snippet
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/Snippet"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	err := ctx.Snippet.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToSnippet(ctx, ctx.Snippet, ctx.Doer))
}

// Get files of a snippet
func GetFiles(ctx *context.APIContext) {
	// swagger:operation GET /snippets/{snippet_uuid}/files snippet getSnippetFiles
	// ---
	// summary: Get files of a Snippet
	// produces:
	// - application/json
	// parameters:
	// - name: snippet_uuid
	//   in: path
	//   description: uuid of the snippet
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/SnippetFiles"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	files, err := snippet_service.GetFiles(ctx, ctx.Snippet)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	ctx.JSON(http.StatusOK, files)
}

// Update files of a Snippet
func UpdateFiles(ctx *context.APIContext) {
	// swagger:operation POST /snippets/{snippet_uuid}/files snippet updateSnippetFiles
	// ---
	// summary: Update files of a Snippet
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: snippet_uuid
	//   in: path
	//   description: uuid of the snippet
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/UpdateSnippetFilesOption"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "400":
	//     "$ref": "#/responses/error"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	//   "422":
	//     "$ref": "#/responses/validationError"
	opt := web.GetForm(ctx).(*api.UpdateSnippetFilesOption)

	files := snippet_service.SnippetFiles(opt.Files)
	err := files.Validate()
	if err != nil {
		ctx.Error(http.StatusBadRequest, err.Error(), nil)
		return
	}

	err = snippet_service.UpdateFiles(ctx, ctx.Snippet, ctx.Doer, opt.Files)
	if err != nil {
		ctx.ServerError("UpdateFiles", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// Deletes a snippet
func Delete(ctx *context.APIContext) {
	// swagger:operation DELETE /snippet/{snippet_uuid} snippet deleteSnippet
	// ---
	// summary: Deletes a Snippet
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: snippet_uuid
	//   in: path
	//   description: uuid of the snippet
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "401":
	//     "$ref": "#/responses/unauthorized"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"
	err := snippet_service.DeleteSnippet(ctx, ctx.Snippet)
	if err != nil {
		ctx.ServerError("DeleteSnippet", err)
		return
	}

	ctx.Status(http.StatusNoContent)
}
