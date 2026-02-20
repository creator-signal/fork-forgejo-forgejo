// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	api "forgejo.org/modules/structs"
)

// ToAPIProject converts a project to its API representation for embedding in issue/PR responses.
// It uses the pre-loaded ProjectColumnID and ProjectColumnTitle from the issue to avoid N+1 queries.
func ToAPIProject(issue *issues_model.Issue, p *project_model.Project) *api.Project {
	result := &api.Project{
		ID:          p.ID,
		Title:       p.Title,
		Description: p.Description,
		State:       p.State(),
		Created:     p.CreatedUnix.AsTime(),
		Updated:     p.UpdatedUnix.AsTimePtr(),
	}
	if p.IsClosed {
		result.Closed = p.ClosedDateUnix.AsTimePtr()
	}

	if issue.ProjectBoardID > 0 {
		result.ColumnID = issue.ProjectBoardID
		result.Column = issue.ProjectColumnName
	}

	return result
}
