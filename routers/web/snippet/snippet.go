// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"fmt"
	"net/http"
	"strings"

	repo_model "forgejo.org/models/repo"
	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/sitemap"
	api "forgejo.org/modules/structs"
	"forgejo.org/routers/common"
	"forgejo.org/services/context"
	snippet_service "forgejo.org/services/snippet"
)

type snippetForm struct {
	Name        string
	Description string
	Visibility  snippet_model.SnippetVisibility
	Files       snippet_service.SnippetFiles
}

// parseSnippetForm parses the form
// This is needed, as the normal parser can't handle the multiple files
func parseSnippetForm(req *http.Request) (*snippetForm, error) {
	err := req.ParseForm()
	if err != nil {
		return nil, err
	}

	form := new(snippetForm)

	form.Name = req.FormValue("name")
	if form.Name == "" {
		return nil, fmt.Errorf("name can't be empty")
	}

	form.Description = req.FormValue("description")

	form.Visibility, err = snippet_model.SnippetVisibilityFromName(req.FormValue("visibility"))
	if err != nil {
		return nil, err
	}

	form.Files = make([]*api.SnippetFile, 0)

	for key, value := range req.Form {
		if !strings.HasPrefix(key, "file-name-") {
			continue
		}

		if len(value) == 0 {
			return nil, fmt.Errorf("%s has no value", key)
		}

		fileID := strings.TrimPrefix(key, "file-name-")

		currentFile := new(api.SnippetFile)
		currentFile.Name = value[0]
		currentFile.Content = req.FormValue(fmt.Sprintf("file-content-%s", fileID))

		form.Files = append(form.Files, currentFile)
	}

	if len(form.Files) == 0 {
		return nil, fmt.Errorf("form has no files")
	}

	return form, nil
}

// New creates a Snippet
func New(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("snippet.edit.new_header")
	ctx.Data["MaxSnippetFiles"] = setting.Snippet.MaxFilesPerSnippet

	ctx.HTML(http.StatusOK, "snippet/add_edit")
}

// NewPost handles the post event for a Snippet page
func NewPost(ctx *context.Context) {
	form, err := parseSnippetForm(ctx.Req)
	if err != nil {
		ctx.ServerError("ParseSnippetForm", err)
		return
	}

	snippet, err := snippet_service.CreateSnippet(ctx, ctx.Doer, form.Name, form.Description, form.Visibility, form.Files)
	if err != nil {
		ctx.ServerError("CreateSnippet", err)
		return
	}

	ctx.Redirect(snippet.Link())
}

// View shows a Snippet
func View(ctx *context.Context) {
	ctx.Snippet.RenderDescription(ctx)

	err := ctx.Snippet.LoadOwner(ctx)
	if err != nil {
		ctx.ServerError("LoadOwner", err)
		return
	}

	files, err := snippet_service.GetFiles(ctx, ctx.Snippet)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	err = files.Highlight()
	if err != nil {
		ctx.ServerError("HighlightFiles", err)
		return
	}

	if ctx.Snippet.Visibility == snippet_model.SnippetVisibilityHidden {
		ctx.Data["AddNoIndexHeader"] = true
	}

	cl := new(repo_model.CloneLink)
	cl.SSH = repo_model.ComposeSSHCloneURL("snippets", ctx.Snippet.UUID)
	cl.HTTPS = repo_model.ComposeHTTPSCloneURL("snippets", ctx.Snippet.UUID)

	ctx.Data["RepoCloneLink"] = cl

	cloneButtonShowHTTPS := !setting.Repository.DisableHTTPGit
	cloneButtonShowSSH := !setting.SSH.Disabled && (ctx.IsSigned || setting.SSH.ExposeAnonymous)
	if !cloneButtonShowHTTPS && !cloneButtonShowSSH {
		// We have to show at least one link, so we just show the HTTPS
		cloneButtonShowHTTPS = true
	}
	ctx.Data["CloneButtonShowHTTPS"] = cloneButtonShowHTTPS
	ctx.Data["CloneButtonShowSSH"] = cloneButtonShowSSH
	ctx.Data["CloneButtonOriginLink"] = ctx.Data["RepoCloneLink"]

	ctx.Data["Snippet"] = ctx.Snippet
	ctx.Data["SnippetFiles"] = files
	ctx.Data["Title"] = ctx.Snippet.Name

	ctx.HTML(http.StatusOK, "snippet/view")
}

func Raw(ctx *context.Context) {
	filename := ctx.Params(":filename")

	blob, err := snippet_service.GetBlob(ctx, ctx.Snippet, filename)
	if err != nil {
		ctx.ServerError("GetBlob", err)
		return
	}

	err = common.ServeBlob(ctx.Base, filename, blob, nil)
	if err != nil {
		ctx.ServerError("ServeBlob", err)
		return
	}
}

// Edit show the edit page
func Edit(ctx *context.Context) {
	files, err := snippet_service.GetFiles(ctx, ctx.Snippet)
	if err != nil {
		ctx.ServerError("GetFiles", err)
		return
	}

	ctx.Data["Snippet"] = ctx.Snippet
	ctx.Data["SnippetFiles"] = files
	ctx.Data["Title"] = ctx.Tr("snippet.edit.edit_header")
	ctx.Data["MaxSnippetFiles"] = setting.Snippet.MaxFilesPerSnippet

	ctx.HTML(http.StatusOK, "snippet/add_edit")
}

// EditPost handles the post for the edit page
func EditPost(ctx *context.Context) {
	form, err := parseSnippetForm(ctx.Req)
	if err != nil {
		ctx.ServerError("ParseSnippetForm", err)
		return
	}

	ctx.Snippet.Name = form.Name
	ctx.Snippet.Description = form.Description
	ctx.Snippet.Visibility = form.Visibility

	err = ctx.Snippet.UpdateCols(ctx, "name", "description", "visibility")
	if err != nil {
		ctx.ServerError("UpdateCols", err)
		return
	}

	files := make(snippet_service.SnippetFiles, 0)
	err = files.Validate()
	if err != nil {
		ctx.ServerError("ValidateNames", err)
		return
	}

	err = snippet_service.UpdateFiles(ctx, ctx.Snippet, ctx.Doer, form.Files)
	if err != nil {
		ctx.ServerError("UpdateFiles", err)
		return
	}

	ctx.Redirect(ctx.Snippet.Link())
}

// Delete deletes a Snippet
func Delete(ctx *context.Context) {
	err := snippet_service.DeleteSnippet(ctx, ctx.Snippet)
	if err != nil {
		ctx.ServerError("DeleteSnippet", err)
		return
	}

	ctx.Flash.Success(ctx.Tr("snippet.delete.success"))

	ctx.Redirect("/")
}

// Sitemap renders the Snippets Sitemap
func Sitemap(ctx *context.Context) {
	opts := new(snippet_model.SearchSnippetsOptions)
	opts.Page = int(ctx.ParamsInt64("idx"))
	opts.PageSize = setting.UI.SitemapPagingNum

	snippets, _, err := snippet_model.SearchSnippets(ctx, opts)
	if err != nil {
		log.Error("Failed to get Snippets: %v", err)
		return
	}

	m := sitemap.NewSitemap()
	for _, item := range snippets {
		m.Add(sitemap.URL{URL: item.HTMLURL(), LastMod: item.UpdatedUnix.AsTimePtr()})
	}

	ctx.Resp.Header().Set("Content-Type", "text/xml")

	if _, err := m.WriteTo(ctx.Resp); err != nil {
		log.Error("Failed writing sitemap: %v", err)
	}
}
