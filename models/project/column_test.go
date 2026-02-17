// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"fmt"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDefaultColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectWithoutDefault, err := GetProjectByID(db.DefaultContext, 5)
	require.NoError(t, err)

	// check if default column was added
	column, err := projectWithoutDefault.GetDefaultColumn(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, int64(5), column.ProjectID)
	assert.Equal(t, "Uncategorized", column.Title)

	projectWithMultipleDefaults, err := GetProjectByID(db.DefaultContext, 6)
	require.NoError(t, err)

	// check if multiple defaults were removed
	column, err = projectWithMultipleDefaults.GetDefaultColumn(db.DefaultContext)
	require.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.Equal(t, int64(9), column.ID)

	// set 8 as default column
	require.NoError(t, SetDefaultColumn(db.DefaultContext, column.ProjectID, 8))

	// then 9 will become a non-default column
	column, err = GetColumn(db.DefaultContext, 9)
	require.NoError(t, err)
	assert.Equal(t, int64(6), column.ProjectID)
	assert.False(t, column.Default)
}

func Test_moveIssuesToAnotherColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column1 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})

	issues, err := column1.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 1, issues[0].ID)

	column2 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 2, ProjectID: 1})
	issues, err = column2.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 1)
	assert.EqualValues(t, 3, issues[0].ID)

	err = column1.moveIssuesToAnotherColumn(db.DefaultContext, column2)
	require.NoError(t, err)

	issues, err = column1.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Empty(t, issues)

	issues, err = column2.GetIssues(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.EqualValues(t, 3, issues[0].ID)
	assert.EqualValues(t, 0, issues[0].Sorting)
	assert.EqualValues(t, 1, issues[1].ID)
	assert.EqualValues(t, 1, issues[1].Sorting)
}

func Test_MoveColumnsOnProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columns, 3)
	assert.EqualValues(t, 0, columns[0].Sorting)
	assert.EqualValues(t, 1, columns[1].Sorting)
	assert.EqualValues(t, 2, columns[2].Sorting)

	err = MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
		0: columns[1].ID,
		1: columns[2].ID,
		2: columns[0].ID,
	})
	require.NoError(t, err)

	columnsAfter, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columnsAfter, 3)
	assert.Equal(t, columns[1].ID, columnsAfter[0].ID)
	assert.Equal(t, columns[2].ID, columnsAfter[1].ID)
	assert.Equal(t, columns[0].ID, columnsAfter[2].ID)
}

func TestMoveColumnsOnProjectSwap(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	require.Len(t, columns, 3)

	// First give them distinct positions
	err = MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
		0: columns[0].ID,
		1: columns[1].ID,
		2: columns[2].ID,
	})
	require.NoError(t, err)

	// Now swap columns 0 and 1 (would collide under single-phase update)
	err = MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
		0: columns[1].ID,
		1: columns[0].ID,
		2: columns[2].ID,
	})
	require.NoError(t, err)

	columnsAfter, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columnsAfter, 3)
	assert.Equal(t, columns[1].ID, columnsAfter[0].ID)
	assert.Equal(t, columns[0].ID, columnsAfter[1].ID)
	assert.Equal(t, columns[2].ID, columnsAfter[2].ID)
}

func TestBatchCountCardsInColumns(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	counts, err := BatchCountCardsInColumns(db.DefaultContext, []int64{1, 2})
	require.NoError(t, err)

	// Verify counts match individual CountCardsInColumn calls
	for _, columnID := range []int64{1, 2} {
		individual, err := CountCardsInColumn(db.DefaultContext, columnID)
		require.NoError(t, err)
		assert.Equal(t, individual, counts[columnID], "count mismatch for column %d", columnID)
	}

	// Empty list should return empty map
	emptyCounts, err := BatchCountCardsInColumns(db.DefaultContext, []int64{})
	require.NoError(t, err)
	assert.Empty(t, emptyCounts)
}

func TestBatchCountProjectColumns(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Test with known project IDs from test fixtures
	projectIDs := []int64{1, 2}
	counts, err := BatchCountProjectColumns(db.DefaultContext, projectIDs)
	require.NoError(t, err)

	// Verify counts match individual GetColumns calls
	for _, pid := range projectIDs {
		project, err := GetProjectByID(db.DefaultContext, pid)
		require.NoError(t, err)
		columns, err := project.GetColumns(db.DefaultContext)
		require.NoError(t, err)
		assert.Equal(t, int64(len(columns)), counts[pid], "count mismatch for project %d", pid)
	}

	// Empty list should return empty map
	emptyCounts, err := BatchCountProjectColumns(db.DefaultContext, []int64{})
	require.NoError(t, err)
	assert.Empty(t, emptyCounts)
}

func TestUpdateColumnSortingZero(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})
	column.Sorting = 5
	require.NoError(t, UpdateColumn(db.DefaultContext, column))

	// Verify it was set to 5
	updated, err := GetColumn(db.DefaultContext, column.ID)
	require.NoError(t, err)
	assert.Equal(t, int8(5), updated.Sorting)

	// Now set it back to 0
	column.Sorting = 0
	require.NoError(t, UpdateColumn(db.DefaultContext, column))

	updated, err = GetColumn(db.DefaultContext, column.ID)
	require.NoError(t, err)
	assert.Equal(t, int8(0), updated.Sorting)
}

func TestUpdateColumnColor(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})

	t.Run("ValidColor", func(t *testing.T) {
		column.Color = "#ff0000"
		require.NoError(t, UpdateColumn(db.DefaultContext, column))

		updated, err := GetColumn(db.DefaultContext, column.ID)
		require.NoError(t, err)
		assert.Equal(t, "#ff0000", updated.Color)
	})

	t.Run("InvalidColor", func(t *testing.T) {
		column.Color = "notacolor"
		err := UpdateColumn(db.DefaultContext, column)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad color code")
	})

	t.Run("ClearColor", func(t *testing.T) {
		column.Color = ""
		require.NoError(t, UpdateColumn(db.DefaultContext, column))

		updated, err := GetColumn(db.DefaultContext, column.ID)
		require.NoError(t, err)
		assert.Empty(t, updated.Color)
	})
}

func TestDeleteColumnByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("DeleteNonDefaultColumn", func(t *testing.T) {
		// Column 2 is non-default in project 1
		column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 2, ProjectID: 1})
		assert.False(t, column.Default)

		// Get issues in column 2 before deletion
		issues, err := column.GetIssues(db.DefaultContext)
		require.NoError(t, err)
		issueCount := len(issues)

		err = DeleteColumnByID(db.DefaultContext, column.ID)
		require.NoError(t, err)

		// Column should not exist anymore
		_, err = GetColumn(db.DefaultContext, 2)
		require.Error(t, err)
		assert.True(t, IsErrProjectColumnNotExist(err))

		// Issues should have moved to the default column
		if issueCount > 0 {
			project := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
			defaultColumn, err := project.GetDefaultColumn(db.DefaultContext)
			require.NoError(t, err)
			defaultIssues, err := defaultColumn.GetIssues(db.DefaultContext)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(defaultIssues), issueCount)
		}
	})

	t.Run("CannotDeleteDefaultColumn", func(t *testing.T) {
		// Column 1 is the default column in project 1
		column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})
		assert.True(t, column.Default)

		err := DeleteColumnByID(db.DefaultContext, column.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete default column")
	})

	t.Run("NonExistentColumnIsSilent", func(t *testing.T) {
		err := DeleteColumnByID(db.DefaultContext, 99999)
		require.NoError(t, err)
	})
}

func TestMoveColumnsOnProjectErrorPaths(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	require.Len(t, columns, 3)

	t.Run("SubsetRejected", func(t *testing.T) {
		// Only include 2 of 3 columns — should fail
		err := MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
			0: columns[0].ID,
			1: columns[1].ID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all columns in the project must be included")
	})

	t.Run("DuplicateColumnIDs", func(t *testing.T) {
		err := MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
			0: columns[0].ID,
			1: columns[0].ID, // duplicate
			2: columns[2].ID,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate column ID")
	})

	t.Run("ColumnFromOtherProject", func(t *testing.T) {
		// Column 5 belongs to project 2
		err := MoveColumnsOnProject(db.DefaultContext, project1, map[int64]int64{
			0: columns[0].ID,
			1: columns[1].ID,
			2: 5, // wrong project
		})
		require.Error(t, err)
	})
}

func TestCreateDefaultColumnsForProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("BasicKanbanTemplate", func(t *testing.T) {
		p := &Project{
			Type:         TypeRepository,
			TemplateType: TemplateTypeBasicKanban,
			CardType:     CardTypeTextOnly,
			Title:        "Kanban Template Test",
			RepoID:       1,
			CreatedUnix:  946684810,
			CreatorID:    2,
		}
		require.NoError(t, NewProject(db.DefaultContext, p))

		columns, err := p.GetColumns(db.DefaultContext)
		require.NoError(t, err)
		require.NotEmpty(t, columns)

		// First column should be the default "Backlog"
		assert.True(t, columns[0].Default)
		assert.Equal(t, "Backlog", columns[0].Title)
		assert.Equal(t, int8(0), columns[0].Sorting)

		// All columns should have unique, sequential sorting values
		for i := 1; i < len(columns); i++ {
			assert.Greater(t, columns[i].Sorting, columns[i-1].Sorting,
				"column %d sorting should be greater than column %d", i, i-1)
		}
	})

	t.Run("NoTemplate", func(t *testing.T) {
		p := &Project{
			Type:         TypeRepository,
			TemplateType: TemplateTypeNone,
			CardType:     CardTypeTextOnly,
			Title:        "No Template Test",
			RepoID:       1,
			CreatedUnix:  946684810,
			CreatorID:    2,
		}
		require.NoError(t, NewProject(db.DefaultContext, p))

		// No template means no default columns are created
		columns, err := p.GetColumns(db.DefaultContext)
		require.NoError(t, err)
		assert.Empty(t, columns)
	})
}

func TestNewColumnSortingAssignment(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	p := &Project{
		Type:        TypeRepository,
		CardType:    CardTypeTextOnly,
		Title:       "Sorting Test Project",
		RepoID:      1,
		CreatedUnix: 946684810,
		CreatorID:   2,
	}
	require.NoError(t, NewProject(db.DefaultContext, p))

	// First column gets sorting 0
	col1 := &Column{Title: "Col 1", ProjectID: p.ID}
	require.NoError(t, NewColumn(db.DefaultContext, col1))
	assert.Equal(t, int8(0), col1.Sorting)

	// Second column gets sorting 1
	col2 := &Column{Title: "Col 2", ProjectID: p.ID}
	require.NoError(t, NewColumn(db.DefaultContext, col2))
	assert.Equal(t, int8(1), col2.Sorting)

	// Third column gets sorting 2
	col3 := &Column{Title: "Col 3", ProjectID: p.ID}
	require.NoError(t, NewColumn(db.DefaultContext, col3))
	assert.Equal(t, int8(2), col3.Sorting)
}

func TestGetColumnsByIDsUnscoped(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("ValidIDs", func(t *testing.T) {
		result, err := GetColumnsByIDsUnscoped(db.DefaultContext, []int64{1, 2, 5})
		require.NoError(t, err)
		assert.Len(t, result, 3)
		assert.NotNil(t, result[1])
		assert.NotNil(t, result[2])
		assert.NotNil(t, result[5])
	})

	t.Run("EmptyInput", func(t *testing.T) {
		result, err := GetColumnsByIDsUnscoped(db.DefaultContext, []int64{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("NonExistentIDs", func(t *testing.T) {
		result, err := GetColumnsByIDsUnscoped(db.DefaultContext, []int64{99998, 99999})
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func Test_NewColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project1 := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
	columns, err := project1.GetColumns(db.DefaultContext)
	require.NoError(t, err)
	assert.Len(t, columns, 3)

	for i := 0; i < maxProjectColumns-3; i++ {
		err := NewColumn(db.DefaultContext, &Column{
			Title:     fmt.Sprintf("column-%d", i+4),
			ProjectID: project1.ID,
		})
		require.NoError(t, err)
	}
	err = NewColumn(db.DefaultContext, &Column{
		Title:     "column-21",
		ProjectID: project1.ID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum number of columns reached")
}
