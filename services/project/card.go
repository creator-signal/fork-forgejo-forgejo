// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"

	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
)

// CardPosition represents a card's position in reordering
type CardPosition struct {
	IssueID int64
	Sorting int64
}

// ReorderCardsInColumn reorders cards within a project column
// This is the shared business logic used by both web UI and API
func ReorderCardsInColumn(ctx context.Context, column *project_model.Column, cardPositions []CardPosition) error {
	if len(cardPositions) == 0 {
		return nil
	}

	// Build arrays for validation
	issueIDs := make([]int64, 0, len(cardPositions))
	sortedIssueIDs := make(map[int64]int64)

	for _, cp := range cardPositions {
		issueIDs = append(issueIDs, cp.IssueID)
		sortedIssueIDs[cp.Sorting] = cp.IssueID
	}

	// Fetch issues for repo-ownership validation
	issues, err := issues_model.GetIssuesByIDs(ctx, issueIDs)
	if err != nil {
		return err
	}

	// Get project to validate repo IDs
	project, err := project_model.GetProjectByID(ctx, column.ProjectID)
	if err != nil {
		return err
	}

	// Validate all issues belong to the project's repository (only for repo projects)
	if project.RepoID > 0 {
		for _, issue := range issues {
			if issue.RepoID != project.RepoID {
				return project_model.ErrCardNotInProjectRepo{IssueID: issue.ID, ProjectRepoID: project.RepoID}
			}
		}
	}

	// Perform the reordering
	return project_model.MoveIssuesOnProjectColumn(ctx, column, sortedIssueIDs)
}

// AddCardToColumn adds an issue to a project column with validation
func AddCardToColumn(ctx context.Context, column *project_model.Column, issueID, sorting int64) (*project_model.ProjectIssue, error) {
	// Validate issue exists and belongs to correct repo
	issue, err := issues_model.GetIssueByID(ctx, issueID)
	if err != nil {
		return nil, err
	}

	project, err := project_model.GetProjectByID(ctx, column.ProjectID)
	if err != nil {
		return nil, err
	}

	// Validate issue belongs to project's repository (only for repo projects)
	if project.RepoID > 0 && issue.RepoID != project.RepoID {
		return nil, project_model.ErrCardNotInProjectRepo{IssueID: issueID, ProjectRepoID: project.RepoID}
	}

	err = project_model.AddIssueToProject(ctx, project.ID, issueID, column.ID, sorting)
	if err != nil {
		return nil, err
	}

	return project_model.GetProjectCard(ctx, project.ID, issueID)
}

// RemoveCardFromProject removes an issue from a project
func RemoveCardFromProject(ctx context.Context, project *project_model.Project, issueID int64) error {
	return project_model.RemoveIssueFromProject(ctx, project.ID, issueID)
}
