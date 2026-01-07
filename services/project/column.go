// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"

	project_model "forgejo.org/models/project"
	user_model "forgejo.org/models/user"
)

// CreateColumnOptions represents options for creating a project column
type CreateColumnOptions struct {
	Title string
	Color string
}

// UpdateColumnOptions represents options for updating a project column
type UpdateColumnOptions struct {
	Title *string
	Color *string
}

// CreateColumn creates a new project column with proper validation
func CreateColumn(ctx context.Context, project *project_model.Project, creator *user_model.User, opts CreateColumnOptions) (*project_model.Column, error) {
	column := &project_model.Column{
		ProjectID: project.ID,
		Title:     opts.Title,
		Color:     opts.Color,
		CreatorID: creator.ID,
	}

	return column, project_model.NewColumn(ctx, column)
}

// UpdateColumn updates an existing project column
func UpdateColumn(ctx context.Context, column *project_model.Column, opts UpdateColumnOptions) error {
	if opts.Title != nil {
		column.Title = *opts.Title
	}
	if opts.Color != nil {
		column.Color = *opts.Color
	}

	return project_model.UpdateColumn(ctx, column)
}

// DeleteColumn deletes a project column with proper cleanup
func DeleteColumn(ctx context.Context, column *project_model.Column) error {
	return project_model.DeleteColumnByID(ctx, column.ID)
}
