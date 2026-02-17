// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"sync"
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReorderCardsInColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	project := &project_model.Project{
		Title:       "Test Project",
		Description: "Test project for card reordering",
		RepoID:      repo.ID,
		CreatorID:   owner.ID,
		Type:        project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	column := &project_model.Column{
		Title:     "Test Column",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, column))

	issue1 := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Test Issue 1",
		Content:  "Test content 1",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue1, nil, nil))

	issue2 := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Test Issue 2",
		Content:  "Test content 2",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue2, nil, nil))

	require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue1.ID, column.ID, 0))
	require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue2.ID, column.ID, 1))

	t.Run("Normal reordering", func(t *testing.T) {
		cardPositions := []CardPosition{
			{IssueID: issue2.ID, Sorting: 0},
			{IssueID: issue1.ID, Sorting: 1},
		}

		err := ReorderCardsInColumn(db.DefaultContext, column, cardPositions)
		require.NoError(t, err)

		cards, _, err := project_model.GetProjectCardsInColumn(db.DefaultContext, column.ID, db.ListOptions{})
		require.NoError(t, err)
		require.Len(t, cards, 2)

		assert.Equal(t, issue2.ID, cards[0].IssueID)
		assert.Equal(t, int64(0), cards[0].Sorting)
		assert.Equal(t, issue1.ID, cards[1].IssueID)
		assert.Equal(t, int64(1), cards[1].Sorting)
	})

	t.Run("Empty card list", func(t *testing.T) {
		err := ReorderCardsInColumn(db.DefaultContext, column, []CardPosition{})
		require.NoError(t, err)
	})

	t.Run("Non-existent issue", func(t *testing.T) {
		cardPositions := []CardPosition{
			{IssueID: 99999, Sorting: 0},
		}

		err := ReorderCardsInColumn(db.DefaultContext, column, cardPositions)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not in column")
	})

	t.Run("Issue from wrong repository", func(t *testing.T) {
		otherRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})

		otherIssue := &issues_model.Issue{
			RepoID:   otherRepo.ID,
			PosterID: owner.ID,
			Title:    "Other Issue",
			Content:  "Issue from different repo",
		}
		require.NoError(t, issues_model.NewIssue(db.DefaultContext, otherRepo, otherIssue, nil, nil))

		cardPositions := []CardPosition{
			{IssueID: otherIssue.ID, Sorting: 0},
		}

		err := ReorderCardsInColumn(db.DefaultContext, column, cardPositions)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "card issue")
	})
}

func TestReorderCardsInColumnWrongColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	project := &project_model.Project{
		Title:     "Wrong Column Test Project",
		RepoID:    repo.ID,
		CreatorID: owner.ID,
		Type:      project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	columnA := &project_model.Column{
		Title:     "Column A",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, columnA))

	columnB := &project_model.Column{
		Title:     "Column B",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, columnB))

	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Wrong Column Issue",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue, nil, nil))

	require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue.ID, columnB.ID, 0))

	cardPositions := []CardPosition{
		{IssueID: issue.ID, Sorting: 0},
	}
	err := ReorderCardsInColumn(db.DefaultContext, columnA, cardPositions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not in column")
}

func TestReorderCardsInColumnDuplicateSorting(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	project := &project_model.Project{
		Title:     "Duplicate Sorting Test Project",
		RepoID:    repo.ID,
		CreatorID: owner.ID,
		Type:      project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	column := &project_model.Column{
		Title:     "Test Column",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, column))

	issue1 := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Dup Sort Issue 1",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue1, nil, nil))

	issue2 := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Dup Sort Issue 2",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue2, nil, nil))

	require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue1.ID, column.ID, 0))
	require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue2.ID, column.ID, 1))

	cardPositions := []CardPosition{
		{IssueID: issue1.ID, Sorting: 0},
		{IssueID: issue2.ID, Sorting: 0},
	}
	err := ReorderCardsInColumn(db.DefaultContext, column, cardPositions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate sorting position")
}

func TestReorderCardsInColumnConcurrent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	project := &project_model.Project{
		Title:       "Concurrent Test Project",
		Description: "Test project for concurrent card reordering",
		RepoID:      repo.ID,
		CreatorID:   owner.ID,
		Type:        project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	column := &project_model.Column{
		Title:     "Concurrent Test Column",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, column))

	var issues []*issues_model.Issue
	for i := 0; i < 5; i++ {
		issue := &issues_model.Issue{
			RepoID:   repo.ID,
			PosterID: owner.ID,
			Title:    "Concurrent Test Issue",
			Content:  "Test content",
		}
		require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue, nil, nil))
		require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue.ID, column.ID, int64(i)))
		issues = append(issues, issue)
	}

	var wg sync.WaitGroup
	var errors []error
	var errorsMutex sync.Mutex

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			cardPositions := make([]CardPosition, len(issues))
			for j, issue := range issues {
				sortPos := int64((j + workerID) % len(issues))
				cardPositions[j] = CardPosition{
					IssueID: issue.ID,
					Sorting: sortPos,
				}
			}

			err := ReorderCardsInColumn(db.DefaultContext, column, cardPositions)
			if err != nil {
				errorsMutex.Lock()
				errors = append(errors, err)
				errorsMutex.Unlock()
			}
		}(i)
	}

	wg.Wait()

	assert.Empty(t, errors, "No errors should occur during concurrent reordering")

	cards, _, err := project_model.GetProjectCardsInColumn(db.DefaultContext, column.ID, db.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, cards, len(issues), "All cards should still be present")

	sortingValues := make(map[int64]bool)
	for _, card := range cards {
		assert.False(t, sortingValues[card.Sorting], "No duplicate sorting values should exist")
		sortingValues[card.Sorting] = true
	}
}

func TestAddCardToColumn(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})
	otherRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})

	project := &project_model.Project{
		Title:       "Add Card Test Project",
		Description: "Test project for adding cards",
		RepoID:      repo.ID,
		CreatorID:   owner.ID,
		Type:        project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	column := &project_model.Column{
		Title:     "Add Card Test Column",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, column))

	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Test Issue for Card Addition",
		Content:  "Test content",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue, nil, nil))

	t.Run("Successfully add card to column", func(t *testing.T) {
		sorting := int64(1)
		card, err := AddCardToColumn(db.DefaultContext, column, issue.ID, sorting)
		require.NoError(t, err)
		require.NotNil(t, card)

		assert.Equal(t, issue.ID, card.IssueID)
		assert.Equal(t, project.ID, card.ProjectID)
		assert.Equal(t, column.ID, card.ProjectColumnID)
		assert.Equal(t, sorting, card.Sorting)
	})

	t.Run("Error when adding duplicate card", func(t *testing.T) {
		_, err := AddCardToColumn(db.DefaultContext, column, issue.ID, int64(2))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists in project")
	})

	t.Run("Error when issue doesn't exist", func(t *testing.T) {
		_, err := AddCardToColumn(db.DefaultContext, column, int64(99999), int64(1))
		require.Error(t, err)
	})

	t.Run("Error when issue from wrong repository", func(t *testing.T) {
		wrongRepoIssue := &issues_model.Issue{
			RepoID:   otherRepo.ID,
			PosterID: owner.ID,
			Title:    "Wrong Repo Issue",
			Content:  "Issue from wrong repo",
		}
		require.NoError(t, issues_model.NewIssue(db.DefaultContext, otherRepo, wrongRepoIssue, nil, nil))

		_, err := AddCardToColumn(db.DefaultContext, column, wrongRepoIssue.ID, int64(1))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not belong to project repository")
	})
}

func TestRemoveCardFromProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	project := &project_model.Project{
		Title:       "Remove Card Test Project",
		Description: "Test project for removing cards",
		RepoID:      repo.ID,
		CreatorID:   owner.ID,
		Type:        project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, project))

	column := &project_model.Column{
		Title:     "Remove Card Test Column",
		ProjectID: project.ID,
	}
	require.NoError(t, project_model.NewColumn(db.DefaultContext, column))

	issue := &issues_model.Issue{
		RepoID:   repo.ID,
		PosterID: owner.ID,
		Title:    "Test Issue for Card Removal",
		Content:  "Test content",
	}
	require.NoError(t, issues_model.NewIssue(db.DefaultContext, repo, issue, nil, nil))

	t.Run("Successfully remove card from project", func(t *testing.T) {
		require.NoError(t, project_model.AddIssueToProject(db.DefaultContext, project.ID, issue.ID, column.ID, -1))

		card, err := project_model.GetProjectCard(db.DefaultContext, project.ID, issue.ID)
		require.NoError(t, err)
		require.NotNil(t, card)

		err = RemoveCardFromProject(db.DefaultContext, project, issue.ID)
		require.NoError(t, err)

		card, err = project_model.GetProjectCard(db.DefaultContext, project.ID, issue.ID)
		require.Error(t, err)
		assert.Nil(t, card)
	})
}
