// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveIssuesOnProjectColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Get column 1 which belongs to project 1 and has issue 1
	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})
	require.Equal(t, int64(1), column.ProjectID)

	t.Run("Success", func(t *testing.T) {
		// Issue 1 is in column 1 (from fixtures)
		sortedIssueIDs := map[int64]int64{
			0: 1, // sorting position 0 -> issue_id 1
		}
		err := MoveIssuesOnProjectColumn(db.DefaultContext, column, sortedIssueIDs)
		require.NoError(t, err)

		// Verify the sorting was updated
		card, err := GetProjectCard(db.DefaultContext, column.ProjectID, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(0), card.Sorting)
	})

	t.Run("MoveIssueFromDifferentColumn", func(t *testing.T) {
		// Issue 3 is in column 2, not column 1 — but same project, so cross-column move should succeed
		sortedIssueIDs := map[int64]int64{
			0: 3,
		}
		err := MoveIssuesOnProjectColumn(db.DefaultContext, column, sortedIssueIDs)
		require.NoError(t, err)

		// Verify the card was moved to column 1 and sorting updated
		card, err := GetProjectCard(db.DefaultContext, column.ProjectID, 3)
		require.NoError(t, err)
		assert.Equal(t, column.ID, card.ProjectColumnID)
		assert.Equal(t, int64(0), card.Sorting)
	})

	t.Run("ErrorIssueNotInProject", func(t *testing.T) {
		// Issue 999 doesn't exist
		sortedIssueIDs := map[int64]int64{
			0: 999,
		}
		err := MoveIssuesOnProjectColumn(db.DefaultContext, column, sortedIssueIDs)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "all issues must belong to the specified project")
	})
}

func TestAddIssueToProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Success", func(t *testing.T) {
		// Add issue 4 to project 1, column 1 (issue 4 is not in any project in fixtures)
		err := AddIssueToProject(db.DefaultContext, 1, 4, 1, -1)
		require.NoError(t, err)

		// Verify it was added
		card, err := GetProjectCard(db.DefaultContext, 1, 4)
		require.NoError(t, err)
		assert.Equal(t, int64(4), card.IssueID)
		assert.Equal(t, int64(1), card.ProjectColumnID)
		// Auto-sorting should have set a value > 0
		assert.GreaterOrEqual(t, card.Sorting, int64(0))
	})

	t.Run("ErrorDuplicate", func(t *testing.T) {
		// Issue 1 is already in project 1 (from fixtures)
		err := AddIssueToProject(db.DefaultContext, 1, 1, 1, -1)
		require.Error(t, err)
		assert.True(t, IsErrCardAlreadyInProject(err))
	})

	t.Run("ErrorColumnNotInProject", func(t *testing.T) {
		// Column 5 belongs to project 2, not project 1
		err := AddIssueToProject(db.DefaultContext, 1, 6, 5, -1)
		require.Error(t, err)
		assert.True(t, IsErrProjectColumnNotExist(err))
	})

	t.Run("ExplicitPositionZero", func(t *testing.T) {
		// When sorting is 0, it should place at position 0 (not auto-append)
		err := AddIssueToProject(db.DefaultContext, 1, 8, 1, 0)
		require.NoError(t, err)

		card, err := GetProjectCard(db.DefaultContext, 1, 8)
		require.NoError(t, err)
		assert.Equal(t, int64(0), card.Sorting)
	})

	t.Run("AutoAppendWithNegative", func(t *testing.T) {
		// When sorting is -1, it should auto-append to end
		err := AddIssueToProject(db.DefaultContext, 1, 7, 1, -1)
		require.NoError(t, err)

		card, err := GetProjectCard(db.DefaultContext, 1, 7)
		require.NoError(t, err)
		assert.Positive(t, card.Sorting)
	})
}

func TestMoveCardToColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Success", func(t *testing.T) {
		// Card 1 (issue 1) is in column 1, move to column 2 (both in project 1)
		card := unittest.AssertExistsAndLoadBean(t, &ProjectIssue{ID: 1})
		require.Equal(t, int64(1), card.ProjectColumnID)

		err := MoveCardToColumn(db.DefaultContext, card.ID, 2, -1)
		require.NoError(t, err)

		// Verify move
		updatedCard, err := GetProjectCard(db.DefaultContext, card.ProjectID, card.IssueID)
		require.NoError(t, err)
		assert.Equal(t, int64(2), updatedCard.ProjectColumnID)
	})

	t.Run("ErrorCardNotFound", func(t *testing.T) {
		err := MoveCardToColumn(db.DefaultContext, 99999, 2, -1)
		require.Error(t, err)
		assert.True(t, IsErrProjectCardNotExist(err))
	})

	t.Run("ErrorColumnNotInSameProject", func(t *testing.T) {
		// Card 3 is in project 1, try to move to column 5 which is in project 2
		card := unittest.AssertExistsAndLoadBean(t, &ProjectIssue{ID: 3})
		require.Equal(t, int64(1), card.ProjectID)

		err := MoveCardToColumn(db.DefaultContext, card.ID, 5, -1)
		require.Error(t, err)
		assert.True(t, IsErrProjectColumnNotExist(err))
	})
}

func TestMoveCardToColumnWithPositionShift(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("InsertAtPositionShiftsOtherCards", func(t *testing.T) {
		// Setup: Create a column with multiple cards at sequential positions
		projectID := int64(1)
		columnID := int64(1)

		// Add test cards with known positions
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 10, columnID, 1))
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 11, columnID, 2))
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 12, columnID, 3))

		// Verify initial positions
		card10, _ := GetProjectCard(db.DefaultContext, projectID, 10)
		card11, _ := GetProjectCard(db.DefaultContext, projectID, 11)
		card12, _ := GetProjectCard(db.DefaultContext, projectID, 12)
		assert.Equal(t, int64(1), card10.Sorting)
		assert.Equal(t, int64(2), card11.Sorting)
		assert.Equal(t, int64(3), card12.Sorting)

		// Move card12 to position 2 - should shift card11 to position 3
		err := MoveCardToColumn(db.DefaultContext, card12.ID, columnID, 2)
		require.NoError(t, err)

		// Verify: card12 should be at position 2, card11 should have shifted to 3
		card10After, _ := GetProjectCard(db.DefaultContext, projectID, 10)
		card11After, _ := GetProjectCard(db.DefaultContext, projectID, 11)
		card12After, _ := GetProjectCard(db.DefaultContext, projectID, 12)

		assert.Equal(t, int64(1), card10After.Sorting, "card10 should remain at position 1")
		assert.Equal(t, int64(2), card12After.Sorting, "card12 should now be at position 2")
		assert.Equal(t, int64(3), card11After.Sorting, "card11 should have shifted to position 3")
	})

	t.Run("MoveToNewColumnWithPosition", func(t *testing.T) {
		// Setup: Add cards to column 2
		projectID := int64(1)
		sourceColumn := int64(1)
		targetColumn := int64(2)

		// Add test cards to target column with known positions
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 20, targetColumn, 1))
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 21, targetColumn, 2))

		// Add a card to source column (auto-append)
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 22, sourceColumn, -1))
		cardToMove, _ := GetProjectCard(db.DefaultContext, projectID, 22)

		// Move card from source to target at position 1
		err := MoveCardToColumn(db.DefaultContext, cardToMove.ID, targetColumn, 1)
		require.NoError(t, err)

		// Verify positions in target column
		card20, _ := GetProjectCard(db.DefaultContext, projectID, 20)
		card21, _ := GetProjectCard(db.DefaultContext, projectID, 21)
		card22, _ := GetProjectCard(db.DefaultContext, projectID, 22)

		assert.Equal(t, int64(1), card22.Sorting, "moved card should be at position 1")
		assert.Equal(t, int64(2), card20.Sorting, "card20 should have shifted to position 2")
		assert.Equal(t, int64(3), card21.Sorting, "card21 should have shifted to position 3")
		assert.Equal(t, targetColumn, card22.ProjectColumnID, "card should be in target column")
	})
}

func TestGetProjectIssueByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("ExistingCard", func(t *testing.T) {
		card := unittest.AssertExistsAndLoadBean(t, &ProjectIssue{ID: 1})
		fetched, err := GetProjectIssueByID(db.DefaultContext, card.ID)
		require.NoError(t, err)
		assert.Equal(t, card.ID, fetched.ID)
		assert.Equal(t, card.IssueID, fetched.IssueID)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := GetProjectIssueByID(db.DefaultContext, 99999)
		require.Error(t, err)
		assert.True(t, IsErrProjectCardNotExist(err))
	})
}

func TestMoveIssuesOnProjectColumnSwap(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})

	// Setup: add two cards at distinct positions
	require.NoError(t, AddIssueToProject(db.DefaultContext, 1, 14, column.ID, 10))
	require.NoError(t, AddIssueToProject(db.DefaultContext, 1, 15, column.ID, 11))

	// Swap them: card at 10→11, card at 11→10
	sortedIssueIDs := map[int64]int64{
		11: 14, // issue 14 goes to position 11
		10: 15, // issue 15 goes to position 10
	}
	err := MoveIssuesOnProjectColumn(db.DefaultContext, column, sortedIssueIDs)
	require.NoError(t, err)

	card14, err := GetProjectCard(db.DefaultContext, 1, 14)
	require.NoError(t, err)
	assert.Equal(t, int64(11), card14.Sorting)

	card15, err := GetProjectCard(db.DefaultContext, 1, 15)
	require.NoError(t, err)
	assert.Equal(t, int64(10), card15.Sorting)
}

func TestShiftCardsForInsertionTwoPhase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Setup: 3 cards at positions 1, 2, 3
	projectID := int64(1)
	columnID := int64(1)

	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 40, columnID, 1))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 41, columnID, 2))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 42, columnID, 3))

	// Insert a new card at position 2 — should shift cards at 2+ by 1
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 43, columnID, 2))

	card40, _ := GetProjectCard(db.DefaultContext, projectID, 40)
	card41, _ := GetProjectCard(db.DefaultContext, projectID, 41)
	card42, _ := GetProjectCard(db.DefaultContext, projectID, 42)
	card43, _ := GetProjectCard(db.DefaultContext, projectID, 43)

	assert.Equal(t, int64(1), card40.Sorting, "card40 should stay at 1")
	assert.Equal(t, int64(2), card43.Sorting, "card43 should be at 2")
	assert.Equal(t, int64(3), card41.Sorting, "card41 should shift to 3")
	assert.Equal(t, int64(4), card42.Sorting, "card42 should shift to 4")
}

func TestAddIssueToProjectWithPositionShift(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("InsertAtPositionShiftsExistingCards", func(t *testing.T) {
		// Setup: Create column with cards at positions 1, 2, 3
		projectID := int64(1)
		columnID := int64(1)

		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 30, columnID, 1))
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 31, columnID, 2))
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 32, columnID, 3))

		// Insert new card at position 2 - should shift cards at positions 2 and 3
		require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 33, columnID, 2))

		// Verify positions
		card30, _ := GetProjectCard(db.DefaultContext, projectID, 30)
		card31, _ := GetProjectCard(db.DefaultContext, projectID, 31)
		card32, _ := GetProjectCard(db.DefaultContext, projectID, 32)
		card33, _ := GetProjectCard(db.DefaultContext, projectID, 33)

		assert.Equal(t, int64(1), card30.Sorting, "card30 should remain at position 1")
		assert.Equal(t, int64(2), card33.Sorting, "new card33 should be at position 2")
		assert.Equal(t, int64(3), card31.Sorting, "card31 should have shifted to position 3")
		assert.Equal(t, int64(4), card32.Sorting, "card32 should have shifted to position 4")
	})
}

func TestRemoveIssueFromProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Success", func(t *testing.T) {
		// Issue 1 is in project 1 (from fixtures)
		card, err := GetProjectCard(db.DefaultContext, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), card.IssueID)

		err = RemoveIssueFromProject(db.DefaultContext, 1, 1)
		require.NoError(t, err)

		// Should no longer exist
		_, err = GetProjectCard(db.DefaultContext, 1, 1)
		require.Error(t, err)
		assert.True(t, IsErrProjectCardNotExist(err))
	})

	t.Run("NonExistentIsSilent", func(t *testing.T) {
		// Removing an issue not in the project should not error
		err := RemoveIssueFromProject(db.DefaultContext, 1, 99999)
		require.NoError(t, err)
	})
}

func TestGetProjectCardsInColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("BasicRetrieval", func(t *testing.T) {
		// Column 1 in project 1 has issue 1
		cards, total, err := GetProjectCardsInColumn(db.DefaultContext, 1, db.ListOptions{})
		require.NoError(t, err)
		assert.Positive(t, total)
		assert.Len(t, cards, int(total))
	})

	t.Run("WithPagination", func(t *testing.T) {
		// Add multiple cards to column 2 for pagination test
		require.NoError(t, AddIssueToProject(db.DefaultContext, 1, 50, 2, 0))
		require.NoError(t, AddIssueToProject(db.DefaultContext, 1, 51, 2, 1))
		require.NoError(t, AddIssueToProject(db.DefaultContext, 1, 52, 2, 2))

		cards, total, err := GetProjectCardsInColumn(db.DefaultContext, 2, db.ListOptions{Page: 1, PageSize: 2})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, total, int64(3))
		assert.LessOrEqual(t, len(cards), 2)
	})

	t.Run("EmptyColumn", func(t *testing.T) {
		// Create a new project with columns (BasicKanban template creates Backlog + template columns)
		p := &Project{
			Type:         TypeRepository,
			TemplateType: TemplateTypeBasicKanban,
			CardType:     CardTypeTextOnly,
			Title:        "Empty Column Test",
			RepoID:       1,
			CreatedUnix:  946684810,
			CreatorID:    2,
		}
		require.NoError(t, NewProject(db.DefaultContext, p))
		cols, err := p.GetColumns(db.DefaultContext)
		require.NoError(t, err)
		require.NotEmpty(t, cols)

		// The columns should be empty since we haven't added any cards
		cards, total, err := GetProjectCardsInColumn(db.DefaultContext, cols[0].ID, db.ListOptions{})
		require.NoError(t, err)
		assert.Equal(t, int64(0), total)
		assert.Empty(t, cards)
	})

	t.Run("OrderBySorting", func(t *testing.T) {
		// Retrieve cards from column 2 and verify they are ordered by sorting
		cards, _, err := GetProjectCardsInColumn(db.DefaultContext, 2, db.ListOptions{})
		require.NoError(t, err)
		for i := 1; i < len(cards); i++ {
			assert.LessOrEqual(t, cards[i-1].Sorting, cards[i].Sorting)
		}
	})
}

func TestGetProjectIssueColumnIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("ExistingIssues", func(t *testing.T) {
		// Issues 1 and 3 are in project 1 with known columns
		result, err := GetProjectIssueColumnIDs(db.DefaultContext, []int64{1, 3})
		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(1), result[1]) // issue 1 is in column 1
		assert.Equal(t, int64(2), result[3]) // issue 3 is in column 2
	})

	t.Run("NonExistentIssues", func(t *testing.T) {
		result, err := GetProjectIssueColumnIDs(db.DefaultContext, []int64{99998, 99999})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("MixedExistence", func(t *testing.T) {
		result, err := GetProjectIssueColumnIDs(db.DefaultContext, []int64{1, 99999})
		require.NoError(t, err)
		assert.Len(t, result, 1)
		assert.Equal(t, int64(1), result[1])
	})
}

func TestCountCardsInColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Column 1 in project 1 has issue 1
	count, err := CountCardsInColumn(db.DefaultContext, 1)
	require.NoError(t, err)
	assert.Positive(t, count)

	// Non-existent column should return 0
	count, err = CountCardsInColumn(db.DefaultContext, 99999)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
}

// NOTE: TestCountProjectIssues and TestBatchCountProjectIssues are not tested here
// because they JOIN with the `issue` table, which requires importing `models/issues`
// to register the table — but that would create an import cycle.
// These functions are tested via integration tests instead.

func TestShiftCardsAtPositionZero(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectID := int64(1)
	columnID := int64(1)

	// Add cards at positions 0, 1, 2
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 60, columnID, 0))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 61, columnID, 1))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 62, columnID, 2))

	// Insert at position 0 — should shift all existing cards
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 63, columnID, 0))

	card60, _ := GetProjectCard(db.DefaultContext, projectID, 60)
	card61, _ := GetProjectCard(db.DefaultContext, projectID, 61)
	card62, _ := GetProjectCard(db.DefaultContext, projectID, 62)
	card63, _ := GetProjectCard(db.DefaultContext, projectID, 63)

	assert.Equal(t, int64(0), card63.Sorting, "new card63 should be at 0")
	assert.Equal(t, int64(1), card60.Sorting, "card60 should shift from 0 to 1")
	assert.Equal(t, int64(2), card61.Sorting, "card61 should shift from 1 to 2")
	assert.Equal(t, int64(3), card62.Sorting, "card62 should shift from 2 to 3")
}

func TestMoveCardWithinSameColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectID := int64(1)
	columnID := int64(3)

	// Setup: 3 cards at positions 0, 1, 2
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 70, columnID, 0))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 71, columnID, 1))
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 72, columnID, 2))

	// Move card72 from position 2 to position 0 (within same column)
	card72, err := GetProjectCard(db.DefaultContext, projectID, 72)
	require.NoError(t, err)

	err = MoveCardToColumn(db.DefaultContext, card72.ID, columnID, 0)
	require.NoError(t, err)

	card70After, _ := GetProjectCard(db.DefaultContext, projectID, 70)
	card71After, _ := GetProjectCard(db.DefaultContext, projectID, 71)
	card72After, _ := GetProjectCard(db.DefaultContext, projectID, 72)

	assert.Equal(t, int64(0), card72After.Sorting, "card72 should be at position 0")
	assert.Equal(t, int64(1), card70After.Sorting, "card70 should shift to position 1")
	assert.Equal(t, int64(2), card71After.Sorting, "card71 should shift to position 2")
	assert.Equal(t, columnID, card72After.ProjectColumnID, "card72 should remain in same column")
}

func TestMoveIssuesOnProjectColumnEmptyMap(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})
	err := MoveIssuesOnProjectColumn(db.DefaultContext, column, map[int64]int64{})
	require.NoError(t, err) // empty map should be a no-op
}

func TestMoveIssuesOnProjectColumnDuplicateIssueIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	column := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1})
	err := MoveIssuesOnProjectColumn(db.DefaultContext, column, map[int64]int64{
		0: 1,
		1: 1, // duplicate issue ID
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate issue IDs")
}

func TestMoveIssuesOnProjectColumnCrossColumnWithExisting(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectID := int64(1)
	columnID := int64(3)

	// Setup: column 3 already has issue 5 at sorting 0 (from fixtures)
	// Add another card to column 3
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 80, columnID, 1))

	// Now move issue 1 (currently in column 1) into column 3 at position 0
	column3 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 3})
	err := MoveIssuesOnProjectColumn(db.DefaultContext, column3, map[int64]int64{
		0: 1, // issue 1 to position 0 (was in column 1)
	})
	require.NoError(t, err)

	// issue 1 should now be in column 3 at position 0
	card1, err := GetProjectCard(db.DefaultContext, projectID, 1)
	require.NoError(t, err)
	assert.Equal(t, columnID, card1.ProjectColumnID)
	assert.Equal(t, int64(0), card1.Sorting)

	// Pre-existing cards (issue 5 and 80) should be repacked after position 0
	card5, err := GetProjectCard(db.DefaultContext, projectID, 5)
	require.NoError(t, err)
	assert.Equal(t, columnID, card5.ProjectColumnID)
	assert.Positive(t, card5.Sorting, "existing card should be shifted after moved cards")
}

func TestShiftCardsForInsertionBeyondEnd(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projectID := int64(1)
	columnID := int64(3)

	// Setup: card at position 0
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 90, columnID, 0))

	// Insert at position 100 — beyond all existing cards, should be a no-op for existing cards
	require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 91, columnID, 100))

	card90, err := GetProjectCard(db.DefaultContext, projectID, 90)
	require.NoError(t, err)
	assert.Equal(t, int64(0), card90.Sorting, "existing card should not have shifted")

	card91, err := GetProjectCard(db.DefaultContext, projectID, 91)
	require.NoError(t, err)
	assert.Equal(t, int64(100), card91.Sorting, "new card should be at position 100")
}

func TestMoveIssuesToAnotherColumnErrorPaths(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("DifferentProject", func(t *testing.T) {
		col1 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})
		col5 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 5, ProjectID: 2})

		err := col1.moveIssuesToAnotherColumn(db.DefaultContext, col5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "columns have to be in the same project")
	})

	t.Run("SameColumnIsNoOp", func(t *testing.T) {
		col1 := unittest.AssertExistsAndLoadBean(t, &Column{ID: 1, ProjectID: 1})

		err := col1.moveIssuesToAnotherColumn(db.DefaultContext, col1)
		require.NoError(t, err)
	})
}
