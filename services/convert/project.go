// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	"context"

	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
)

// ToAPIProject converts a project to its API representation
func ToAPIProject(ctx context.Context, p *project_model.Project) *api.Project {
	apiProject := &api.Project{
		ID:           p.ID,
		Title:        p.Title,
		Body:         p.Description,
		State:        p.State(),
		TemplateType: api.ProjectTemplateType(p.TemplateType),
		CardType:     api.ProjectCardType(p.CardType),
		Type:         api.ProjectType(p.Type),
		Created:      p.CreatedUnix.AsTime(),
		Updated:      p.UpdatedUnix.AsTimePtr(),
	}

	if p.IsClosed {
		apiProject.Closed = p.ClosedDateUnix.AsTimePtr()
	}

	// Get counts
	if columns, err := p.GetColumns(ctx); err == nil {
		apiProject.ColumnCount = len(columns)

		// Count issues in this project
		openCount, closedCount := countProjectIssues(ctx, p.ID)
		apiProject.OpenIssues = openCount
		apiProject.ClosedIssues = closedCount
	}

	return apiProject
}

// ToAPIProjectList converts a slice of projects to their API representation
func ToAPIProjectList(ctx context.Context, projects []*project_model.Project) []*api.Project {
	result := make([]*api.Project, len(projects))
	for i, project := range projects {
		result[i] = ToAPIProject(ctx, project)
	}
	return result
}

// ToAPIProjectColumn converts a project column to its API representation
func ToAPIProjectColumn(ctx context.Context, col *project_model.Column) *api.ProjectColumn {
	apiColumn := &api.ProjectColumn{
		ID:      col.ID,
		Title:   col.Title,
		Color:   col.Color,
		Default: col.Default,
		Sorting: col.Sorting,
		Created: col.CreatedUnix.AsTime(),
		Updated: col.UpdatedUnix.AsTimePtr(),
	}

	// Count cards in this column
	if count, err := countCardsInColumn(ctx, col.ID); err == nil {
		apiColumn.CardCount = int(count)
	}

	return apiColumn
}

// ToAPIProjectColumnList converts a slice of project columns to their API representation
func ToAPIProjectColumnList(ctx context.Context, columns []*project_model.Column) []*api.ProjectColumn {
	result := make([]*api.ProjectColumn, len(columns))
	for i, column := range columns {
		result[i] = ToAPIProjectColumn(ctx, column)
	}
	return result
}

// ToAPIProjectCard converts a project issue to its API representation
func ToAPIProjectCard(ctx context.Context, doer *user_model.User, projectIssue *project_model.ProjectIssue, issue *issues_model.Issue) *api.ProjectCard {
	apiCard := &api.ProjectCard{
		ID:      projectIssue.ID,
		Sorting: projectIssue.Sorting,
		Issue:   ToIssue(ctx, doer, issue),
	}

	// Load and populate the column
	if projectIssue.ProjectColumnID > 0 {
		if column, err := project_model.GetColumn(ctx, projectIssue.ProjectColumnID); err == nil {
			apiCard.Column = ToAPIProjectColumn(ctx, column)
		}
	}

	// Load and populate the project
	if projectIssue.ProjectID > 0 {
		if project, err := project_model.GetProjectByID(ctx, projectIssue.ProjectID); err == nil {
			apiCard.Project = ToAPIProject(ctx, project)
		}
	}

	return apiCard
}

// ToAPIProjectCardList converts a slice of project issues to their API representation
func ToAPIProjectCardList(ctx context.Context, doer *user_model.User, projectIssues []*project_model.ProjectIssue, issues []*issues_model.Issue) []*api.ProjectCard {
	// Create map for quick issue lookup
	issueMap := make(map[int64]*issues_model.Issue)
	for _, issue := range issues {
		issueMap[issue.ID] = issue
	}

	result := make([]*api.ProjectCard, 0, len(projectIssues))
	for _, projectIssue := range projectIssues {
		if issue, exists := issueMap[projectIssue.IssueID]; exists {
			result = append(result, ToAPIProjectCard(ctx, doer, projectIssue, issue))
		}
	}
	return result
}

// Helper functions for counting

func countProjectIssues(ctx context.Context, projectID int64) (open, closed int) {
	openCount, closedCount, err := project_model.CountProjectIssues(ctx, projectID)
	if err != nil {
		return 0, 0
	}
	return int(openCount), int(closedCount)
}

func countCardsInColumn(ctx context.Context, columnID int64) (int64, error) {
	return project_model.CountCardsInColumn(ctx, columnID)
}
