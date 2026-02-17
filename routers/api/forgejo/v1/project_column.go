// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"net/http"

	project_model "forgejo.org/models/project"
	"forgejo.org/modules/json"
	project_service "forgejo.org/services/project"
)

// ── Repo Project Columns ────────────────────────────────────────

func (f *Forgejo) ListRepoProjectColumns(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireRepoProjectReader(w, r, repo, ctx.Doer) {
		return
	}

	project := loadRepoProject(w, r, repo.ID, projectID)
	if project == nil {
		return
	}

	columns, err := project.GetColumns(r.Context())
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectColumnList(r.Context(), columns))
}

func (f *Forgejo) CreateRepoProjectColumn(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireRepoProjectWriter(w, r, repo, ctx.Doer) {
		return
	}

	project := loadRepoProject(w, r, repo.ID, projectID)
	if project == nil {
		return
	}

	var body CreateColumnOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		apiError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	opts := project_service.CreateColumnOptions{
		Title: body.Title,
	}
	if body.Color != nil {
		opts.Color = *body.Color
	}

	column, err := project_service.CreateColumn(r.Context(), project, ctx.Doer, opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProjectColumn(r.Context(), column))
}

func (f *Forgejo) MoveRepoProjectColumns(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireRepoProjectWriter(w, r, repo, ctx.Doer) {
		return
	}

	project := loadRepoProject(w, r, repo.ID, projectID)
	if project == nil {
		return
	}

	var body MoveColumnsOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if len(body.Columns) == 0 {
		apiError(w, http.StatusUnprocessableEntity, "columns list is required")
		return
	}

	sortedColumnIDs := make(map[int64]int64, len(body.Columns))
	for _, cp := range body.Columns {
		sortedColumnIDs[int64(cp.Position)] = cp.ColumnId
	}

	if err := project_model.MoveColumnsOnProject(r.Context(), project, sortedColumnIDs); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (f *Forgejo) UpdateRepoProjectColumn(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, columnID Column) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireRepoProjectWriter(w, r, repo, ctx.Doer) {
		return
	}

	project := loadRepoProject(w, r, repo.ID, projectID)
	if project == nil {
		return
	}
	col := loadColumn(w, r, project.ID, columnID)
	if col == nil {
		return
	}

	var body EditColumnOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	opts := project_service.UpdateColumnOptions{
		Title: body.Title,
		Color: body.Color,
	}

	if err := project_service.UpdateColumn(r.Context(), col, opts); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectColumn(r.Context(), col))
}

func (f *Forgejo) DeleteRepoProjectColumn(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, columnID Column) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireRepoProjectWriter(w, r, repo, ctx.Doer) {
		return
	}

	project := loadRepoProject(w, r, repo.ID, projectID)
	if project == nil {
		return
	}
	col := loadColumn(w, r, project.ID, columnID)
	if col == nil {
		return
	}

	if err := project_service.DeleteColumn(r.Context(), col); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Org Project Columns ─────────────────────────────────────────

func (f *Forgejo) ListOrgProjectColumns(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
	org, _ := loadOrg(w, r, orgName)
	if org == nil {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}

	columns, err := project.GetColumns(r.Context())
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectColumnList(r.Context(), columns))
}

func (f *Forgejo) CreateOrgProjectColumn(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
	org, ctx := loadOrg(w, r, orgName)
	if org == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireOrgMember(w, r, org, ctx.Doer) {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}

	var body CreateColumnOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		apiError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	opts := project_service.CreateColumnOptions{
		Title: body.Title,
	}
	if body.Color != nil {
		opts.Color = *body.Color
	}

	column, err := project_service.CreateColumn(r.Context(), project, ctx.Doer, opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProjectColumn(r.Context(), column))
}

func (f *Forgejo) MoveOrgProjectColumns(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
	org, ctx := loadOrg(w, r, orgName)
	if org == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireOrgMember(w, r, org, ctx.Doer) {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}

	var body MoveColumnsOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if len(body.Columns) == 0 {
		apiError(w, http.StatusUnprocessableEntity, "columns list is required")
		return
	}

	sortedColumnIDs := make(map[int64]int64, len(body.Columns))
	for _, cp := range body.Columns {
		sortedColumnIDs[int64(cp.Position)] = cp.ColumnId
	}

	if err := project_model.MoveColumnsOnProject(r.Context(), project, sortedColumnIDs); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (f *Forgejo) UpdateOrgProjectColumn(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, columnID Column) {
	org, ctx := loadOrg(w, r, orgName)
	if org == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireOrgMember(w, r, org, ctx.Doer) {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}
	col := loadColumn(w, r, project.ID, columnID)
	if col == nil {
		return
	}

	var body EditColumnOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	opts := project_service.UpdateColumnOptions{
		Title: body.Title,
		Color: body.Color,
	}

	if err := project_service.UpdateColumn(r.Context(), col, opts); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectColumn(r.Context(), col))
}

func (f *Forgejo) DeleteOrgProjectColumn(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, columnID Column) {
	org, ctx := loadOrg(w, r, orgName)
	if org == nil {
		return
	}
	if !requireAuth(w, ctx) {
		return
	}
	if !requireOrgOwner(w, r, org, ctx.Doer) {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}
	col := loadColumn(w, r, project.ID, columnID)
	if col == nil {
		return
	}

	if err := project_service.DeleteColumn(r.Context(), col); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
