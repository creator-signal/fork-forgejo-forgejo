// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package v1

import (
	"context"
	"time"

	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
)

func strPtr(s string) *string        { return &s }
func timePtr(t time.Time) *time.Time { return &t }
func int64Ptr(i int64) *int64        { return &i }

// toAPIProject converts a project model to the API type
func toAPIProject(ctx context.Context, p *project_model.Project) *Project {
	open, closed, _ := project_model.CountProjectIssues(ctx, p.ID)
	colCounts, _ := project_model.BatchCountProjectColumns(ctx, []int64{p.ID})

	state := ProjectStateOpen
	if p.IsClosed {
		state = ProjectStateClosed
	}

	templateType := int(p.TemplateType)
	cardType := int(p.CardType)
	projectType := int(p.Type)
	openIssues := int(open)
	closedIssues := int(closed)
	colCount := int(colCounts[p.ID])

	result := &Project{
		Id:           &p.ID,
		Title:        strPtr(p.Title),
		Body:         strPtr(p.Description),
		State:        &state,
		TemplateType: &templateType,
		CardType:     &cardType,
		Type:         &projectType,
		CreatedAt:    timePtr(p.CreatedUnix.AsTime()),
		UpdatedAt:    timePtr(p.UpdatedUnix.AsTime()),
		ColumnCount:  &colCount,
		OpenIssues:   &openIssues,
		ClosedIssues: &closedIssues,
	}

	if p.IsClosed && int64(p.ClosedDateUnix) > 0 {
		result.ClosedAt = timePtr(p.ClosedDateUnix.AsTime())
	}

	return result
}

// toAPIProjectList converts a list of project models using batch queries to avoid N+1
func toAPIProjectList(ctx context.Context, projects []*project_model.Project) []Project {
	if len(projects) == 0 {
		return []Project{}
	}

	projectIDs := make([]int64, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
	}

	issueCounts, _ := project_model.BatchCountProjectIssues(ctx, projectIDs)
	columnCounts, _ := project_model.BatchCountProjectColumns(ctx, projectIDs)

	result := make([]Project, len(projects))
	for i, p := range projects {
		state := ProjectStateOpen
		if p.IsClosed {
			state = ProjectStateClosed
		}

		templateType := int(p.TemplateType)
		cardType := int(p.CardType)
		projectType := int(p.Type)

		ic := issueCounts[p.ID]
		openIssues := int(ic.Open)
		closedIssues := int(ic.Closed)
		colCount := int(columnCounts[p.ID])

		result[i] = Project{
			Id:           &p.ID,
			Title:        strPtr(p.Title),
			Body:         strPtr(p.Description),
			State:        &state,
			TemplateType: &templateType,
			CardType:     &cardType,
			Type:         &projectType,
			CreatedAt:    timePtr(p.CreatedUnix.AsTime()),
			UpdatedAt:    timePtr(p.UpdatedUnix.AsTime()),
			ColumnCount:  &colCount,
			OpenIssues:   &openIssues,
			ClosedIssues: &closedIssues,
		}

		if p.IsClosed && int64(p.ClosedDateUnix) > 0 {
			result[i].ClosedAt = timePtr(p.ClosedDateUnix.AsTime())
		}
	}

	return result
}

// toAPIProjectColumn converts a column model to the API type
func toAPIProjectColumn(ctx context.Context, col *project_model.Column) *ProjectColumn {
	cardCount, _ := project_model.CountCardsInColumn(ctx, col.ID)

	sorting := int(col.Sorting)
	cc := int(cardCount)

	return &ProjectColumn{
		Id:        &col.ID,
		Title:     strPtr(col.Title),
		Color:     strPtr(col.Color),
		Default:   &col.Default,
		Sorting:   &sorting,
		CreatedAt: timePtr(col.CreatedUnix.AsTime()),
		UpdatedAt: timePtr(col.UpdatedUnix.AsTime()),
		CardCount: &cc,
	}
}

// toAPIProjectColumnList converts a list of columns using batch queries
func toAPIProjectColumnList(ctx context.Context, columns []*project_model.Column) []ProjectColumn {
	if len(columns) == 0 {
		return []ProjectColumn{}
	}

	columnIDs := make([]int64, len(columns))
	for i, c := range columns {
		columnIDs[i] = c.ID
	}
	cardCounts, _ := project_model.BatchCountCardsInColumns(ctx, columnIDs)

	result := make([]ProjectColumn, len(columns))
	for i, col := range columns {
		sorting := int(col.Sorting)
		cc := int(cardCounts[col.ID])
		result[i] = ProjectColumn{
			Id:        &col.ID,
			Title:     strPtr(col.Title),
			Color:     strPtr(col.Color),
			Default:   &col.Default,
			Sorting:   &sorting,
			CreatedAt: timePtr(col.CreatedUnix.AsTime()),
			UpdatedAt: timePtr(col.UpdatedUnix.AsTime()),
			CardCount: &cc,
		}
	}
	return result
}

// toAPIProjectCard converts a project issue (card) model to the API type
func toAPIProjectCard(ctx context.Context, pi *project_model.ProjectIssue) *ProjectCard {
	card := &ProjectCard{
		Id:      &pi.ID,
		Sorting: &pi.Sorting,
	}

	if issue, err := issues_model.GetIssueByID(ctx, pi.IssueID); err == nil {
		issueState := "open"
		if issue.IsClosed {
			issueState = "closed"
		}
		card.Issue = &ProjectCardIssue{
			Id:     &issue.ID,
			Number: int64Ptr(issue.Index),
			Title:  strPtr(issue.Title),
			State:  strPtr(issueState),
			IsPull: &issue.IsPull,
		}
	}

	if col, err := project_model.GetColumn(ctx, pi.ProjectColumnID); err == nil {
		card.Column = toAPIProjectColumn(ctx, col)
	}

	return card
}

// toAPIProjectCardList converts a list of project cards using batch queries
func toAPIProjectCardList(ctx context.Context, cards []*project_model.ProjectIssue) []ProjectCard {
	if len(cards) == 0 {
		return []ProjectCard{}
	}

	// Batch load issues
	issueIDs := make([]int64, len(cards))
	columnIDSet := make(map[int64]bool)
	for i, c := range cards {
		issueIDs[i] = c.IssueID
		columnIDSet[c.ProjectColumnID] = true
	}

	issueMap := make(map[int64]*issues_model.Issue)
	if issues, err := issues_model.GetIssuesByIDs(ctx, issueIDs); err == nil {
		for _, iss := range issues {
			issueMap[iss.ID] = iss
		}
	}

	// Batch load columns
	colIDSlice := make([]int64, 0, len(columnIDSet))
	for id := range columnIDSet {
		colIDSlice = append(colIDSlice, id)
	}
	colMap, _ := project_model.GetColumnsByIDsUnscoped(ctx, colIDSlice)
	cardCountMap, _ := project_model.BatchCountCardsInColumns(ctx, colIDSlice)

	result := make([]ProjectCard, len(cards))
	for i, pi := range cards {
		result[i] = ProjectCard{
			Id:      &pi.ID,
			Sorting: &pi.Sorting,
		}

		if issue, ok := issueMap[pi.IssueID]; ok {
			issueState := "open"
			if issue.IsClosed {
				issueState = "closed"
			}
			result[i].Issue = &ProjectCardIssue{
				Id:     &issue.ID,
				Number: int64Ptr(issue.Index),
				Title:  strPtr(issue.Title),
				State:  strPtr(issueState),
				IsPull: &issue.IsPull,
			}
		}

		if col, ok := colMap[pi.ProjectColumnID]; ok {
			sorting := int(col.Sorting)
			cc := int(cardCountMap[col.ID])
			result[i].Column = &ProjectColumn{
				Id:        &col.ID,
				Title:     strPtr(col.Title),
				Color:     strPtr(col.Color),
				Default:   &col.Default,
				Sorting:   &sorting,
				CreatedAt: timePtr(col.CreatedUnix.AsTime()),
				UpdatedAt: timePtr(col.UpdatedUnix.AsTime()),
				CardCount: &cc,
			}
		}
	}
	return result
}
