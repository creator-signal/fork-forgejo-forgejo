// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	snippet_service "forgejo.org/services/snippet"
)

const (
	tplSnippetsList base.TplName = "admin/snippet/list"
)

// Snippets shows all snippets
func Snippets(ctx *context.Context) {
	opts := new(snippet_model.SearchSnippetsOptions)

	opts.Actor = ctx.Doer
	opts.PageSize = setting.UI.ExplorePagingNum

	opts.Page = ctx.FormInt("page")
	if opts.Page <= 0 {
		opts.Page = 1
	}

	sortOrder := ctx.FormString("sort")
	if sortOrder == "" {
		sortOrder = setting.UI.ExploreDefaultSort
	}
	ctx.Data["SortType"] = sortOrder
	opts.SortOrder = sortOrder

	opts.Keyword = ctx.FormTrim("q")

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

	ctx.Data["Title"] = ctx.Tr("admin.snippets.title")
	ctx.Data["PageIsAdminSnippets"] = true
	ctx.Data["Snippets"] = snippets
	ctx.Data["Total"] = count

	pager := context.NewPagination(int(count), setting.UI.PackagesPagingNum, opts.Page, 5)
	pager.AddParamString("q", opts.Keyword)
	pager.AddParamString("sort", opts.SortOrder)
	ctx.Data["Page"] = pager

	ctx.HTML(http.StatusOK, tplSnippetsList)
}

// DeleteSnippet deletes a snippet
func DeleteSnippet(ctx *context.Context) {
	snippet, err := snippet_model.GetSnippetByUUID(ctx, ctx.FormString("id"))
	if err != nil {
		ctx.ServerError("GetSnippetByUUID", err)
		return
	}

	err = snippet_service.DeleteSnippet(ctx, snippet)
	if err != nil {
		ctx.ServerError("DeleteSnippet", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("snippet.delete.success"))

	ctx.JSONRedirect("/admin/snippets")
}
