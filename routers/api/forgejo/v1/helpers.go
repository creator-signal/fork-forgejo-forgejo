// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"net/http"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/optional"
	"forgejo.org/services/context"
)

// getAPIContext extracts the Forgejo APIContext from the request.
// The shared middleware stack (shared.Middlewares) already injects it.
func getAPIContext(r *http.Request) *context.APIContext {
	return context.GetAPIContext(r)
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// apiError writes a JSON error response
func apiError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, APIError{Message: &message})
}

// requireAuth ensures the user is authenticated
func requireAuth(w http.ResponseWriter, ctx *context.APIContext) bool {
	if !ctx.IsSigned {
		apiError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	return true
}

// loadRepo loads the repository from owner/repo path params and validates access.
// Returns the repo or nil (with error already written to response).
func loadRepo(w http.ResponseWriter, r *http.Request, ownerName, repoName string) (*repo_model.Repository, *context.APIContext) {
	ctx := getAPIContext(r)

	owner, err := user_model.GetUserByName(r.Context(), ownerName)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			apiError(w, http.StatusNotFound, "owner not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil, ctx
	}

	repo, err := repo_model.GetRepositoryByName(r.Context(), owner.ID, repoName)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			apiError(w, http.StatusNotFound, "repository not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil, ctx
	}
	repo.Owner = owner

	// Check access
	perm, err := access_model.GetUserRepoPermission(r.Context(), repo, ctx.Doer)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return nil, ctx
	}
	if !perm.HasAccess() {
		apiError(w, http.StatusNotFound, "repository not found")
		return nil, ctx
	}

	return repo, ctx
}

// requireRepoProjectReader checks that the user can read the Projects unit
func requireRepoProjectReader(w http.ResponseWriter, r *http.Request, repo *repo_model.Repository, doer *user_model.User) bool {
	if !repo.UnitEnabled(r.Context(), unit.TypeProjects) {
		apiError(w, http.StatusNotFound, "projects not enabled for this repository")
		return false
	}
	perm, err := access_model.GetUserRepoPermission(r.Context(), repo, doer)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !perm.CanRead(unit.TypeProjects) {
		apiError(w, http.StatusForbidden, "insufficient permissions")
		return false
	}
	return true
}

// requireRepoProjectWriter checks that the user can write to the Projects unit
func requireRepoProjectWriter(w http.ResponseWriter, r *http.Request, repo *repo_model.Repository, doer *user_model.User) bool {
	if !repo.UnitEnabled(r.Context(), unit.TypeProjects) {
		apiError(w, http.StatusNotFound, "projects not enabled for this repository")
		return false
	}
	p, err := access_model.GetUserRepoPermission(r.Context(), repo, doer)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !p.CanWrite(unit.TypeProjects) {
		apiError(w, http.StatusForbidden, "insufficient permissions")
		return false
	}
	return true
}

// canWriteRepoProjects checks if the user has write access to the project unit
func canWriteRepoProjects(r *http.Request, repo *repo_model.Repository, doer *user_model.User) bool {
	p, err := access_model.GetUserRepoPermission(r.Context(), repo, doer)
	if err != nil {
		return false
	}
	return p.CanWrite(unit.TypeProjects) || p.AccessMode >= perm.AccessModeAdmin
}

// loadOrg loads the organization from the org path param.
// Returns the org or nil (with error already written to response).
func loadOrg(w http.ResponseWriter, r *http.Request, orgName string) (*organization.Organization, *context.APIContext) {
	ctx := getAPIContext(r)

	org, err := organization.GetOrgByName(r.Context(), orgName)
	if err != nil {
		if organization.IsErrOrgNotExist(err) {
			apiError(w, http.StatusNotFound, "organization not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil, ctx
	}

	return org, ctx
}

// requireOrgMember checks that the user is an org member
func requireOrgMember(w http.ResponseWriter, r *http.Request, org *organization.Organization, doer *user_model.User) bool {
	if doer == nil {
		apiError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if doer.IsAdmin {
		return true
	}
	isMember, err := organization.IsOrganizationMember(r.Context(), org.ID, doer.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !isMember {
		apiError(w, http.StatusForbidden, "must be an organization member")
		return false
	}
	return true
}

// requireOrgOwner checks that the user is an org owner
func requireOrgOwner(w http.ResponseWriter, r *http.Request, org *organization.Organization, doer *user_model.User) bool {
	if doer == nil {
		apiError(w, http.StatusUnauthorized, "authentication required")
		return false
	}
	if doer.IsAdmin {
		return true
	}
	isOwner, err := organization.IsOrganizationOwner(r.Context(), org.ID, doer.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return false
	}
	if !isOwner {
		apiError(w, http.StatusForbidden, "must be an organization owner")
		return false
	}
	return true
}

// buildSearchOptions converts query params to project search options
func buildSearchOptions(state, keyword, sort string) (project_model.SearchOptions, bool) {
	opts := project_model.SearchOptions{}

	switch strings.ToLower(state) {
	case "closed":
		opts.IsClosed = optional.Some(true)
	case "all":
		// no filter
	case "open", "":
		opts.IsClosed = optional.Some(false)
	default:
		return opts, false
	}

	if keyword != "" {
		opts.Title = keyword
	}

	if sort != "" {
		if !project_model.IsValidSortType(sort) {
			return opts, false
		}
		opts.OrderBy = project_model.GetSearchOrderByBySortType(sort)
	} else {
		opts.OrderBy = project_model.GetSearchOrderByBySortType("newest")
	}

	return opts, true
}

// paginationFromParams extracts page/limit from query params
func paginationFromParams(page, limit *int) db.ListOptions {
	p := 1
	l := 20
	if page != nil && *page > 0 {
		p = *page
	}
	if limit != nil && *limit > 0 {
		l = *limit
		if l > 50 {
			l = 50
		}
	}
	return db.ListOptions{Page: p, PageSize: l}
}

// loadRepoProject loads and validates a project belongs to a repo
func loadRepoProject(w http.ResponseWriter, r *http.Request, repoID, projectID int64) *project_model.Project {
	p, err := project_model.GetProjectForRepoByID(r.Context(), repoID, projectID)
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			apiError(w, http.StatusNotFound, "project not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil
	}
	return p
}

// loadOrgProject loads and validates a project belongs to an org
func loadOrgProject(w http.ResponseWriter, r *http.Request, orgID, projectID int64) *project_model.Project {
	p, err := project_model.GetProjectForOrgByID(r.Context(), orgID, projectID)
	if err != nil {
		if project_model.IsErrProjectNotExist(err) {
			apiError(w, http.StatusNotFound, "project not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil
	}
	return p
}

// loadColumn loads a column and validates it belongs to the project
func loadColumn(w http.ResponseWriter, r *http.Request, projectID, columnID int64) *project_model.Column {
	col, err := project_model.GetColumn(r.Context(), columnID)
	if err != nil {
		if project_model.IsErrProjectColumnNotExist(err) {
			apiError(w, http.StatusNotFound, "column not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil
	}
	if col.ProjectID != projectID {
		apiError(w, http.StatusNotFound, "column not found in this project")
		return nil
	}
	return col
}

// loadCard loads a card and validates it belongs to the project
func loadCard(w http.ResponseWriter, r *http.Request, projectID, cardID int64) *project_model.ProjectIssue {
	card, err := project_model.GetProjectIssueByID(r.Context(), cardID)
	if err != nil {
		if project_model.IsErrProjectCardNotExist(err) {
			apiError(w, http.StatusNotFound, "card not found")
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return nil
	}
	if card.ProjectID != projectID {
		apiError(w, http.StatusNotFound, "card not found in this project")
		return nil
	}
	return card
}
