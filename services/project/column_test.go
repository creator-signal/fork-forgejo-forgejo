// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"forgejo.org/models/db"
	project_model "forgejo.org/models/project"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	project := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1})

	t.Run("CreateColumn", func(t *testing.T) {
		opts := CreateColumnOptions{
			Title: "Test Column",
			Color: "#ff0000",
		}

		column, err := CreateColumn(db.DefaultContext, project, user, opts)
		require.NoError(t, err)
		assert.NotNil(t, column)
		assert.Equal(t, "Test Column", column.Title)
		assert.Equal(t, "#ff0000", column.Color)
		assert.Equal(t, user.ID, column.CreatorID)
		assert.Equal(t, project.ID, column.ProjectID)
	})
}

func TestUpdateColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Get an existing column
	column := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: 1})
	originalTitle := column.Title
	originalColor := column.Color

	t.Run("UpdateColumnTitle", func(t *testing.T) {
		newTitle := "Updated Column Title"
		opts := UpdateColumnOptions{
			Title: &newTitle,
		}

		err := UpdateColumn(db.DefaultContext, column, opts)
		require.NoError(t, err)
		assert.Equal(t, newTitle, column.Title)
	})

	t.Run("UpdateColumnColor", func(t *testing.T) {
		newColor := "#00ff00"
		opts := UpdateColumnOptions{
			Color: &newColor,
		}

		err := UpdateColumn(db.DefaultContext, column, opts)
		require.NoError(t, err)
		assert.Equal(t, newColor, column.Color)
	})

	// Restore original state
	restoreOpts := UpdateColumnOptions{
		Title: &originalTitle,
		Color: &originalColor,
	}
	_ = UpdateColumn(db.DefaultContext, column, restoreOpts)
}

func TestDeleteColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	project := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1})

	t.Run("Successfully delete column", func(t *testing.T) {
		// Create a new column to delete
		opts := CreateColumnOptions{
			Title: "Column to Delete",
			Color: "#ff0000",
		}

		column, err := CreateColumn(db.DefaultContext, project, user, opts)
		require.NoError(t, err)
		require.NotNil(t, column)

		columnID := column.ID

		// Verify column exists
		foundColumn, err := project_model.GetColumn(db.DefaultContext, columnID)
		require.NoError(t, err)
		require.NotNil(t, foundColumn)

		// Delete the column
		err = DeleteColumn(db.DefaultContext, column)
		require.NoError(t, err)

		// Verify column is deleted
		deletedColumn, err := project_model.GetColumn(db.DefaultContext, columnID)
		require.Error(t, err)
		assert.Nil(t, deletedColumn)
	})
}
