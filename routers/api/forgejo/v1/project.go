// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"fmt"
	"net/http"

	"forgejo.org/models/db"
	project_model "forgejo.org/models/project"
	"forgejo.org/modules/json"
	project_service "forgejo.org/services/project"
)

// ── Repo Projects ───────────────────────────────────────────────

func (f *Forgejo) ListRepoProjects(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, params ListRepoProjectsParams) {
	repo, ctx := loadRepo(w, r, owner, repoName)
	if repo == nil {
		return
	}
	if !requireRepoProjectReader(w, r, repo, ctx.Doer) {
		return
	}

	state := ""
	if params.State != nil {
		state = string(*params.State)
	}
	keyword := ""
	if params.Keyword != nil {
		keyword = *params.Keyword
	}
	sort := ""
	if params.Sort != nil {
		sort = string(*params.Sort)
	}

	opts, ok := buildSearchOptions(state, keyword, sort)
	if !ok {
		apiError(w, http.StatusUnprocessableEntity, "invalid query parameters")
		return
	}
	opts.RepoID = repo.ID
	opts.Type = project_model.TypeRepository
	opts.ListOptions = paginationFromParams(params.Page, params.Limit)

	projects, count, err := db.FindAndCount[project_model.Project](r.Context(), opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", count))
	writeJSON(w, http.StatusOK, toAPIProjectList(r.Context(), projects))
}

func (f *Forgejo) CreateRepoProject(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo) {
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

	var body CreateProjectOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		apiError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	templateType := project_model.TemplateTypeNone
	if body.TemplateType != nil {
		templateType = project_model.TemplateType(*body.TemplateType)
	}

	opts := project_service.CreateProjectOptions{
		Title:        body.Title,
		TemplateType: templateType,
		CanWrite:     canWriteRepoProjects(r, repo, ctx.Doer),
	}
	if body.Body != nil {
		opts.Description = *body.Body
	}

	project, err := project_service.CreateProject(r.Context(), repo, ctx.Doer, opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProject(r.Context(), project))
}

func (f *Forgejo) GetRepoProject(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
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

	writeJSON(w, http.StatusOK, toAPIProject(r.Context(), project))
}

func (f *Forgejo) UpdateRepoProject(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
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

	var body EditProjectOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	opts := project_service.UpdateProjectOptions{
		Title:       body.Title,
		Description: body.Body,
	}
	if body.CardType != nil {
		ct := project_model.CardType(*body.CardType)
		opts.CardType = &ct
	}
	if body.State != nil {
		closed := *body.State == EditProjectOptionStateClosed
		opts.IsClosed = &closed
	}

	if err := project_service.UpdateProject(r.Context(), project, opts); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProject(r.Context(), project))
}

func (f *Forgejo) DeleteRepoProject(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId) {
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

	if err := project_service.DeleteProject(r.Context(), project); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Org Projects ────────────────────────────────────────────────

func (f *Forgejo) ListOrgProjects(w http.ResponseWriter, r *http.Request, orgName Org, params ListOrgProjectsParams) {
	org, _ := loadOrg(w, r, orgName)
	if org == nil {
		return
	}

	state := ""
	if params.State != nil {
		state = string(*params.State)
	}
	keyword := ""
	if params.Keyword != nil {
		keyword = *params.Keyword
	}
	sort := ""
	if params.Sort != nil {
		sort = string(*params.Sort)
	}

	opts, ok := buildSearchOptions(state, keyword, sort)
	if !ok {
		apiError(w, http.StatusUnprocessableEntity, "invalid query parameters")
		return
	}
	opts.OwnerID = org.ID
	opts.Type = project_model.TypeOrganization
	opts.ListOptions = paginationFromParams(params.Page, params.Limit)

	projects, count, err := db.FindAndCount[project_model.Project](r.Context(), opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", count))
	writeJSON(w, http.StatusOK, toAPIProjectList(r.Context(), projects))
}

func (f *Forgejo) CreateOrgProject(w http.ResponseWriter, r *http.Request, orgName Org) {
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

	var body CreateProjectOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.Title == "" {
		apiError(w, http.StatusUnprocessableEntity, "title is required")
		return
	}

	templateType := project_model.TemplateTypeNone
	if body.TemplateType != nil {
		templateType = project_model.TemplateType(*body.TemplateType)
	}

	opts := project_service.CreateProjectOptions{
		Title:        body.Title,
		TemplateType: templateType,
		CanWrite:     true, // org members always have write access to org projects
	}
	if body.Body != nil {
		opts.Description = *body.Body
	}

	project, err := project_service.CreateOrgProject(r.Context(), org, ctx.Doer, opts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProject(r.Context(), project))
}

func (f *Forgejo) GetOrgProject(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
	org, _ := loadOrg(w, r, orgName)
	if org == nil {
		return
	}

	project := loadOrgProject(w, r, org.ID, projectID)
	if project == nil {
		return
	}

	writeJSON(w, http.StatusOK, toAPIProject(r.Context(), project))
}

func (f *Forgejo) UpdateOrgProject(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
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

	var body EditProjectOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	opts := project_service.UpdateProjectOptions{
		Title:       body.Title,
		Description: body.Body,
	}
	if body.CardType != nil {
		ct := project_model.CardType(*body.CardType)
		opts.CardType = &ct
	}
	if body.State != nil {
		closed := *body.State == EditProjectOptionStateClosed
		opts.IsClosed = &closed
	}

	if err := project_service.UpdateProject(r.Context(), project, opts); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProject(r.Context(), project))
}

func (f *Forgejo) DeleteOrgProject(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId) {
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

	if err := project_service.DeleteProject(r.Context(), project); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
