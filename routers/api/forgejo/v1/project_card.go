// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"fmt"
	"net/http"

	project_model "forgejo.org/models/project"
	"forgejo.org/modules/json"
	project_service "forgejo.org/services/project"
)

// ── Repo Project Cards ──────────────────────────────────────────

func (f *Forgejo) ListRepoProjectCards(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, columnID Column, params ListRepoProjectCardsParams) {
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
	col := loadColumn(w, r, project.ID, columnID)
	if col == nil {
		return
	}

	listOpts := paginationFromParams(params.Page, params.Limit)
	cards, count, err := project_model.GetProjectCardsInColumn(r.Context(), col.ID, listOpts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", count))
	writeJSON(w, http.StatusOK, toAPIProjectCardList(r.Context(), cards))
}

func (f *Forgejo) AddRepoProjectCard(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, columnID Column) {
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

	var body AddCardOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.IssueId == 0 {
		apiError(w, http.StatusUnprocessableEntity, "issue_id is required")
		return
	}

	var sorting int64
	if body.Position != nil {
		sorting = *body.Position
	}

	card, err := project_service.AddCardToColumn(r.Context(), col, body.IssueId, sorting)
	if err != nil {
		if project_model.IsErrCardNotInProjectRepo(err) {
			apiError(w, http.StatusUnprocessableEntity, err.Error())
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProjectCard(r.Context(), card))
}

func (f *Forgejo) ReorderRepoProjectCards(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, columnID Column) {
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

	var body ReorderCardsOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if len(body.CardPositions) == 0 {
		apiError(w, http.StatusUnprocessableEntity, "card_positions is required")
		return
	}

	positions := make([]project_service.CardPosition, len(body.CardPositions))
	for i, cp := range body.CardPositions {
		positions[i] = project_service.CardPosition{
			IssueID: cp.CardId,
			Sorting: cp.Position,
		}
	}

	if err := project_service.ReorderCardsInColumn(r.Context(), col, positions); err != nil {
		apiError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (f *Forgejo) MoveRepoProjectCard(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, cardID Card) {
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
	card := loadCard(w, r, project.ID, cardID)
	if card == nil {
		return
	}

	var body MoveCardOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	newColumnID := card.ProjectColumnID
	if body.ColumnId != nil {
		// Validate target column belongs to same project
		targetCol := loadColumn(w, r, project.ID, *body.ColumnId)
		if targetCol == nil {
			return
		}
		newColumnID = targetCol.ID
	}

	var newSorting int64 = -1
	if body.Position != nil {
		newSorting = *body.Position
	}

	if err := project_model.MoveCardToColumn(r.Context(), card.ID, newColumnID, newSorting); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Re-read card after move
	updated, err := project_model.GetProjectIssueByID(r.Context(), card.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectCard(r.Context(), updated))
}

func (f *Forgejo) DeleteRepoProjectCard(w http.ResponseWriter, r *http.Request, owner Owner, repoName Repo, projectID ProjectId, cardID Card) {
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
	card := loadCard(w, r, project.ID, cardID)
	if card == nil {
		return
	}

	if err := project_service.RemoveCardFromProject(r.Context(), project, card.IssueID); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── Org Project Cards ───────────────────────────────────────────

func (f *Forgejo) ListOrgProjectCards(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, columnID Column, params ListOrgProjectCardsParams) {
	org, _ := loadOrg(w, r, orgName)
	if org == nil {
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

	listOpts := paginationFromParams(params.Page, params.Limit)
	cards, count, err := project_model.GetProjectCardsInColumn(r.Context(), col.ID, listOpts)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("X-Total-Count", fmt.Sprintf("%d", count))
	writeJSON(w, http.StatusOK, toAPIProjectCardList(r.Context(), cards))
}

func (f *Forgejo) AddOrgProjectCard(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, columnID Column) {
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

	var body AddCardOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if body.IssueId == 0 {
		apiError(w, http.StatusUnprocessableEntity, "issue_id is required")
		return
	}

	var sorting int64
	if body.Position != nil {
		sorting = *body.Position
	}

	card, err := project_service.AddCardToColumn(r.Context(), col, body.IssueId, sorting)
	if err != nil {
		if project_model.IsErrCardNotInProjectRepo(err) {
			apiError(w, http.StatusUnprocessableEntity, err.Error())
		} else {
			apiError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, toAPIProjectCard(r.Context(), card))
}

func (f *Forgejo) ReorderOrgProjectCards(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, columnID Column) {
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

	var body ReorderCardsOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}
	if len(body.CardPositions) == 0 {
		apiError(w, http.StatusUnprocessableEntity, "card_positions is required")
		return
	}

	positions := make([]project_service.CardPosition, len(body.CardPositions))
	for i, cp := range body.CardPositions {
		positions[i] = project_service.CardPosition{
			IssueID: cp.CardId,
			Sorting: cp.Position,
		}
	}

	if err := project_service.ReorderCardsInColumn(r.Context(), col, positions); err != nil {
		apiError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (f *Forgejo) MoveOrgProjectCard(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, cardID Card) {
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
	card := loadCard(w, r, project.ID, cardID)
	if card == nil {
		return
	}

	var body MoveCardOption
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		apiError(w, http.StatusUnprocessableEntity, "invalid request body")
		return
	}

	newColumnID := card.ProjectColumnID
	if body.ColumnId != nil {
		targetCol := loadColumn(w, r, project.ID, *body.ColumnId)
		if targetCol == nil {
			return
		}
		newColumnID = targetCol.ID
	}

	var newSorting int64 = -1
	if body.Position != nil {
		newSorting = *body.Position
	}

	if err := project_model.MoveCardToColumn(r.Context(), card.ID, newColumnID, newSorting); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	updated, err := project_model.GetProjectIssueByID(r.Context(), card.ID)
	if err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, toAPIProjectCard(r.Context(), updated))
}

func (f *Forgejo) DeleteOrgProjectCard(w http.ResponseWriter, r *http.Request, orgName Org, projectID ProjectId, cardID Card) {
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
	card := loadCard(w, r, project.ID, cardID)
	if card == nil {
		return
	}

	if err := project_service.RemoveCardFromProject(r.Context(), project, card.IssueID); err != nil {
		apiError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
