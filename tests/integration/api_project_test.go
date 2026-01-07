// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIProjects(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// Test creating a project
	createOpt := structs.CreateProjectOption{
		Title:        "test project",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, createOpt.Title, apiProject.Title)
	assert.Equal(t, createOpt.Body, apiProject.Body)
	assert.Equal(t, structs.StateOpen, apiProject.State)
	assert.Equal(t, createOpt.TemplateType, apiProject.TemplateType)

	projectID := apiProject.ID

	// Test getting the project by ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiProject2 structs.Project
	DecodeJSON(t, resp, &apiProject2)
	assert.Equal(t, apiProject.Title, apiProject2.Title)
	assert.Equal(t, apiProject.ID, apiProject2.ID)

	// Test getting the project by title
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%s", owner.Name, repo.Name, apiProject.Title)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiProject3 structs.Project
	DecodeJSON(t, resp, &apiProject3)
	assert.Equal(t, apiProject.Title, apiProject3.Title)
	assert.Equal(t, apiProject.ID, apiProject3.ID)

	// Test updating the project
	updateOpt := structs.EditProjectOption{
		Title: &[]string{"updated project"}[0],
		Body:  &[]string{"updated description"}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID), updateOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, *updateOpt.Title, apiProject.Title)
	assert.Equal(t, *updateOpt.Body, apiProject.Body)

	// Test closing the project
	closedState := structs.StateClosed
	updateOpt2 := structs.EditProjectOption{
		State: &closedState,
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID), updateOpt2).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, structs.StateClosed, apiProject.State)

	// Test that closed_at is set when project is closed
	if apiProject.Closed == nil {
		t.Errorf("Expected closed_at to be set when project is closed, but it was nil")
	} else {
		// Check that closed_at is recent (within last 10 seconds)
		timeDiff := time.Since(*apiProject.Closed)
		if timeDiff > 10*time.Second || timeDiff < 0 {
			t.Errorf("Expected closed_at to be recent, but it was %v ago. Closed at: %v", timeDiff, *apiProject.Closed)
		}
	}

	// Test listing projects
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiProjects []structs.Project
	DecodeJSON(t, resp, &apiProjects)
	found := false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.False(t, found, "closed project should not appear in default (open) listing")

	// Test listing all projects
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects?state=all", owner.Name, repo.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	found = false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found, "closed project should appear in 'all' listing")

	// Test listing closed projects
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects?state=closed", owner.Name, repo.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	found = false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found, "closed project should appear in 'closed' listing")

	// Test reopening the project
	openState := structs.StateOpen
	updateOpt3 := structs.EditProjectOption{
		State: &openState,
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID), updateOpt3).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, structs.StateOpen, apiProject.State)

	// Test that closed_at is cleared when project is reopened
	if apiProject.Closed != nil {
		t.Errorf("Expected closed_at to be cleared when project is reopened, but it was %v", *apiProject.Closed)
	}

	// Test search by title
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects?state=all&q=updated", owner.Name, repo.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	found = false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found, "project should be found by title search")

	// Test sorting - verify projects are returned in alphabetical order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects?state=all&sort=alphabetically", owner.Name, repo.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	assert.NotEmpty(t, apiProjects, "should return projects")
	// Verify alphabetical ordering
	for i := 1; i < len(apiProjects); i++ {
		assert.LessOrEqual(t, apiProjects[i-1].Title, apiProjects[i].Title,
			"projects should be sorted alphabetically: %s should come before or equal to %s",
			apiProjects[i-1].Title, apiProjects[i].Title)
	}

	// Test deleting the project
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify project is deleted
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIProjectPermissions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

	// Test unauthorized creation (read-only token)
	createOpt := structs.CreateProjectOption{
		Title:        "unauthorized test",
		Body:         "test",
		TemplateType: structs.ProjectTemplateTypeNone,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// Test reading (should work with read token)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)
}

func TestAPIProjectColumns(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteIssue)

	// Create a project first
	createOpt := structs.CreateProjectOption{
		Title:        "test project for columns",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// List project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns, "project should have default columns")

	columnID := columns[0].ID

	// Test listing cards in column (should be empty initially)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var cards []structs.ProjectCard
	DecodeJSON(t, resp, &cards)
	assert.Empty(t, cards, "new column should have no cards")

	// Create an issue to add as a card
	issueOpt := structs.CreateIssueOption{
		Title: "test issue for project card",
		Body:  "test issue body",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name), issueOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var issue structs.Issue
	DecodeJSON(t, resp, &issue)

	// Add the issue as a card to the column
	addCardOpt := structs.AddCardToColumnOption{
		IssueID: issue.ID,
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID), addCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var card structs.ProjectCard
	DecodeJSON(t, resp, &card)
	require.NotNil(t, card.Issue, "card should have an issue")
	assert.Equal(t, issue.ID, card.Issue.ID)

	// Test listing cards in column (should now have one card)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &cards)
	require.Len(t, cards, 1, "column should have one card")
	require.NotNil(t, cards[0].Issue, "card should have issue reference")
	assert.Equal(t, issue.ID, cards[0].Issue.ID)
	require.NotNil(t, cards[0].Project, "card should have project reference")
	assert.Equal(t, projectID, cards[0].Project.ID)
	require.NotNil(t, cards[0].Column, "card should have column reference")
	assert.Equal(t, columnID, cards[0].Column.ID)

	// Test with invalid column ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/99999/cards", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with invalid project ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/99999/columns/%d/cards", owner.Name, repo.Name, columnID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIProjectCardReordering(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteIssue)

	// Create a project
	createOpt := structs.CreateProjectOption{
		Title:        "test project for card reordering",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// Get project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns, "project should have default columns")

	columnID := columns[0].ID

	// Create multiple issues to add as cards
	issues := make([]structs.Issue, 3)
	cards := make([]structs.ProjectCard, 3)

	for i := 0; i < 3; i++ {
		issueOpt := structs.CreateIssueOption{
			Title: fmt.Sprintf("test issue %d for card reordering", i+1),
			Body:  "test issue body",
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name), issueOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &issues[i])

		// Add each issue as a card to the column
		addCardOpt := structs.AddCardToColumnOption{
			IssueID: issues[i].ID,
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID), addCardOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &cards[i])
	}

	// Get current card order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var originalCards []structs.ProjectCard
	DecodeJSON(t, resp, &originalCards)
	assert.Len(t, originalCards, 3, "column should have three cards")

	// Reorder cards: reverse the order
	reorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{
			{CardID: originalCards[2].ID, Position: 0}, // Move last card to first
			{CardID: originalCards[1].ID, Position: 1}, // Keep middle card in middle
			{CardID: originalCards[0].ID, Position: 2}, // Move first card to last
		},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify new order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var reorderedCards []structs.ProjectCard
	DecodeJSON(t, resp, &reorderedCards)
	assert.Len(t, reorderedCards, 3, "column should still have three cards")

	// Check that the order is reversed
	assert.Equal(t, originalCards[2].ID, reorderedCards[0].ID, "first card should be the original third card")
	assert.Equal(t, originalCards[1].ID, reorderedCards[1].ID, "second card should be the original second card")
	assert.Equal(t, originalCards[0].ID, reorderedCards[2].ID, "third card should be the original first card")

	// Test error cases

	// Test with invalid card ID
	invalidReorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{
			{CardID: 99999, Position: 0},
		},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), invalidReorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with empty card positions
	emptyReorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), emptyReorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusBadRequest)

	// Test with invalid column ID
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/99999/cards/reorder", owner.Name, repo.Name, projectID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with invalid project ID
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/99999/columns/%d/cards/reorder", owner.Name, repo.Name, columnID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

// TestAPIProjectCardConcurrentReordering tests concurrent card reordering scenarios
func TestAPIProjectCardConcurrentReordering(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, OwnerID: owner.ID})

	// Create access token with proper scopes
	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteIssue)

	// Create a project
	createProjectOpt := structs.CreateProjectOption{
		Title: "Concurrent Test Project",
		Body:  "Project for testing concurrent operations",
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createProjectOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var project structs.Project
	DecodeJSON(t, resp, &project)
	projectID := project.ID

	// Create a column for testing
	createColumnOpt := structs.CreateProjectColumnOption{
		Title: "Test Column",
		Color: "#84c5f4",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID), createColumnOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var column structs.ProjectColumn
	DecodeJSON(t, resp, &column)
	columnID := column.ID

	// Create multiple issues to add as cards
	cardCount := 10
	var issueIDs []int64

	for i := 0; i < cardCount; i++ {
		issueOpt := structs.CreateIssueOption{
			Title: fmt.Sprintf("Concurrent Test Issue %d", i+1),
			Body:  fmt.Sprintf("Issue body %d for concurrent testing", i+1),
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name), issueOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		var issue structs.Issue
		DecodeJSON(t, resp, &issue)
		issueIDs = append(issueIDs, issue.ID)

		// Add issue as card to column
		cardOpt := structs.AddCardToColumnOption{
			IssueID: issue.ID,
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID), cardOpt).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)
	}

	// Get initial card order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var initialCards []structs.ProjectCard
	DecodeJSON(t, resp, &initialCards)
	assert.Len(t, initialCards, cardCount, "column should have all cards")

	// Test concurrent reordering operations
	const numConcurrentOperations = 5
	var wg sync.WaitGroup
	results := make([]error, numConcurrentOperations)

	// Each goroutine will try to reorder cards differently
	for i := 0; i < numConcurrentOperations; i++ {
		wg.Add(1)
		go func(operationID int) {
			defer wg.Done()

			// Create different reordering patterns for each operation
			var cardPositions []structs.CardPosition
			for j, card := range initialCards {
				// Each operation uses a different position calculation
				newPosition := int64((j + operationID*2) % len(initialCards))
				cardPositions = append(cardPositions, structs.CardPosition{
					CardID:   card.ID,
					Position: newPosition,
				})
			}

			reorderOpt := structs.ReorderCardsInColumnOption{
				CardPositions: cardPositions,
			}

			req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), reorderOpt).
				AddTokenAuth(token)

			// Some operations may succeed, others may fail due to concurrent modifications
			// We care more about data consistency than operation success
			defer func() {
				if r := recover(); r != nil {
					results[operationID] = fmt.Errorf("operation %d panicked: %v", operationID, r)
				}
			}()

			resp := MakeRequestNilResponseRecorder(t, req, http.StatusNoContent)
			if resp.Code != http.StatusNoContent {
				results[operationID] = fmt.Errorf("operation %d failed with status %d", operationID, resp.Code)
			}
		}(i)
	}

	// Wait for all concurrent operations to complete
	wg.Wait()

	// Validate concurrent operation results - at least some should succeed
	successCount := 0
	for _, err := range results {
		if err == nil {
			successCount++
		}
	}
	assert.Positive(t, successCount, "at least one concurrent operation should succeed")

	// Verify data consistency after concurrent operations
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var finalCards []structs.ProjectCard
	DecodeJSON(t, resp, &finalCards)

	// Check data consistency
	assert.Len(t, finalCards, cardCount, "all cards should still be present after concurrent operations")

	// Verify no duplicate cards
	cardIDs := make(map[int64]bool)
	for _, card := range finalCards {
		assert.False(t, cardIDs[card.ID], "card ID %d should not appear twice", card.ID)
		cardIDs[card.ID] = true
	}

	// Verify all original cards are still present
	for _, originalCard := range initialCards {
		assert.True(t, cardIDs[originalCard.ID], "original card ID %d should still be present", originalCard.ID)
	}

	// Note: Concurrent operations may result in non-unique sorting values,
	// which is acceptable - cards are still sorted correctly by value

	// Test rapid sequential reordering to ensure transactional integrity
	for i := 0; i < 3; i++ {
		// Reverse the order
		var cardPositions []structs.CardPosition
		for j := len(finalCards) - 1; j >= 0; j-- {
			cardPositions = append(cardPositions, structs.CardPosition{
				CardID:   finalCards[j].ID,
				Position: int64(len(finalCards) - 1 - j),
			})
		}

		reorderOpt := structs.ReorderCardsInColumnOption{
			CardPositions: cardPositions,
		}

		req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), reorderOpt).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// Verify order after each operation
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &finalCards)
		assert.Len(t, finalCards, cardCount, "card count should remain consistent")
	}

	// Test concurrent operations with overlapping card sets
	wg = sync.WaitGroup{}
	const numOverlapOperations = 3

	for i := 0; i < numOverlapOperations; i++ {
		wg.Add(1)
		go func(operationID int) {
			defer wg.Done()

			// Each operation reorders a subset of cards with some overlap
			startIdx := operationID * 2
			endIdx := startIdx + 5
			if endIdx > len(finalCards) {
				endIdx = len(finalCards)
			}
			if startIdx >= len(finalCards) {
				startIdx = len(finalCards) - 3
				endIdx = len(finalCards)
			}

			var cardPositions []structs.CardPosition
			for j := startIdx; j < endIdx; j++ {
				// Shuffle positions within the subset
				newPosition := int64(startIdx + ((j - startIdx + 1) % (endIdx - startIdx)))
				cardPositions = append(cardPositions, structs.CardPosition{
					CardID:   finalCards[j].ID,
					Position: newPosition,
				})
			}

			if len(cardPositions) > 0 {
				reorderOpt := structs.ReorderCardsInColumnOption{
					CardPositions: cardPositions,
				}

				req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards/reorder", owner.Name, repo.Name, projectID, columnID), reorderOpt).
					AddTokenAuth(token)
				// Allow failure in concurrent scenario - we're testing data consistency, not operation success
				MakeRequestNilResponseRecorder(t, req, http.StatusNoContent)
			}
		}(i)
	}

	wg.Wait()

	// Final consistency check
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &finalCards)

	assert.Len(t, finalCards, cardCount, "final card count should match original")

	// Verify no data corruption - card IDs must be unique, issues must exist
	// Note: sorting values may not be unique after concurrent operations, which is acceptable
	finalCardIDs := make(map[int64]bool)
	for _, card := range finalCards {
		assert.False(t, finalCardIDs[card.ID], "final check: card ID %d should not appear twice", card.ID)
		finalCardIDs[card.ID] = true
		require.NotNil(t, card.Issue, "card should have associated issue")
		assert.Positive(t, card.Issue.ID, "card issue should have valid ID")
	}

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIOrganizationProjects(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 owns org3

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	// Test creating a project
	createOpt := structs.CreateProjectOption{
		Title:        "test org project",
		Body:         "test organization project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, createOpt.Title, apiProject.Title)
	assert.Equal(t, createOpt.Body, apiProject.Body)
	assert.Equal(t, structs.StateOpen, apiProject.State)
	assert.Equal(t, createOpt.TemplateType, apiProject.TemplateType)

	projectID := apiProject.ID

	// Test getting the project by ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiProject2 structs.Project
	DecodeJSON(t, resp, &apiProject2)
	assert.Equal(t, apiProject.Title, apiProject2.Title)
	assert.Equal(t, apiProject.ID, apiProject2.ID)

	// Test updating the project
	updateOpt := structs.EditProjectOption{
		Title: &[]string{"updated org project"}[0],
		Body:  &[]string{"updated organization description"}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID), updateOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, *updateOpt.Title, apiProject.Title)
	assert.Equal(t, *updateOpt.Body, apiProject.Body)

	// Test closing the project
	closedState := structs.StateClosed
	updateOpt2 := structs.EditProjectOption{
		State: &closedState,
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID), updateOpt2).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProject)
	assert.Equal(t, structs.StateClosed, apiProject.State)

	// Test listing projects
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var apiProjects []structs.Project
	DecodeJSON(t, resp, &apiProjects)
	found := false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.False(t, found, "closed project should not appear in default (open) listing")

	// Test listing all projects
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects?state=all", org.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	found = false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found, "closed project should appear in 'all' listing")

	// Test search by title
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects?state=all&q=updated", org.Name)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &apiProjects)
	found = false
	for _, p := range apiProjects {
		if p.ID == projectID {
			found = true
			break
		}
	}
	assert.True(t, found, "project should be found by title search")

	// Test deleting the project
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify project is deleted
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIOrganizationProjectPermissions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	// Use a user who is NOT a member of org3 (members are: 1, 2, 4, 5, 28)
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10}) // user10 is not a member of org3

	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadOrganization)

	// Test unauthorized creation (user is not org member)
	createOpt := structs.CreateProjectOption{
		Title:        "unauthorized org test",
		Body:         "test",
		TemplateType: structs.ProjectTemplateTypeNone,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// Test reading (should also be forbidden for non-members)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}

func TestAPIOrganizationProjectColumns(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 owns org3

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	// Create a project first
	createOpt := structs.CreateProjectOption{
		Title:        "test org project for columns",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// List project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns", org.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns, "project should have default columns")

	// Create a new column
	createColumnOpt := structs.CreateProjectColumnOption{
		Title: "New Test Column",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns", org.Name, projectID), createColumnOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var newColumn structs.ProjectColumn
	DecodeJSON(t, resp, &newColumn)
	assert.Equal(t, createColumnOpt.Title, newColumn.Title)

	columnID := newColumn.ID

	// Update the column
	updateColumnOpt := structs.EditProjectColumnOption{
		Title: &[]string{"Updated Column Title"}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d", org.Name, projectID, columnID), updateColumnOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var updatedColumn structs.ProjectColumn
	DecodeJSON(t, resp, &updatedColumn)
	assert.Equal(t, *updateColumnOpt.Title, updatedColumn.Title)

	// Delete the column
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d", org.Name, projectID, columnID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Clean up project
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIProjectColumnReordering(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	// Create a project
	createOpt := structs.CreateProjectOption{
		Title:        "test project for column reordering",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeNone, // No default columns
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)
	projectID := apiProject.ID

	// Create 3 columns
	columnIDs := make([]int64, 3)
	for i := 0; i < 3; i++ {
		createColumnOpt := structs.CreateProjectColumnOption{
			Title: fmt.Sprintf("Column %d", i+1),
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID), createColumnOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		var column structs.ProjectColumn
		DecodeJSON(t, resp, &column)
		columnIDs[i] = column.ID
	}

	// Get initial column order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var originalColumns []structs.ProjectColumn
	DecodeJSON(t, resp, &originalColumns)
	assert.Len(t, originalColumns, 3, "project should have three columns")

	// Verify initial order
	assert.Equal(t, "Column 1", originalColumns[0].Title)
	assert.Equal(t, "Column 2", originalColumns[1].Title)
	assert.Equal(t, "Column 3", originalColumns[2].Title)

	// Reorder columns: reverse the order
	moveOpt := structs.MoveProjectColumnsOption{
		Columns: []structs.ColumnPosition{
			{ColumnID: columnIDs[2], Position: 0}, // Move Column 3 to first
			{ColumnID: columnIDs[1], Position: 1}, // Keep Column 2 in middle
			{ColumnID: columnIDs[0], Position: 2}, // Move Column 1 to last
		},
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/move", owner.Name, repo.Name, projectID), moveOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify new order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var reorderedColumns []structs.ProjectColumn
	DecodeJSON(t, resp, &reorderedColumns)
	assert.Len(t, reorderedColumns, 3, "project should still have three columns")

	// Check that the order is reversed
	assert.Equal(t, "Column 3", reorderedColumns[0].Title, "first column should be the original third column")
	assert.Equal(t, "Column 2", reorderedColumns[1].Title, "second column should be the original second column")
	assert.Equal(t, "Column 1", reorderedColumns[2].Title, "third column should be the original first column")

	// Test error case: empty columns list
	emptyMoveOpt := structs.MoveProjectColumnsOption{
		Columns: []structs.ColumnPosition{},
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/move", owner.Name, repo.Name, projectID), emptyMoveOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusBadRequest)

	// Test error case: invalid project ID
	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/99999/columns/move", owner.Name, repo.Name), moveOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIOrganizationProjectCards(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 owns org3

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization, auth_model.AccessTokenScopeWriteIssue)

	// Create a project first
	createOpt := structs.CreateProjectOption{
		Title:        "test org project for cards",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// Get project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns", org.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns, "project should have default columns")

	columnID := columns[0].ID

	// Test listing cards in column (should be empty initially)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var cards []structs.ProjectCard
	DecodeJSON(t, resp, &cards)
	assert.Empty(t, cards, "new column should have no cards")

	// Create an issue to add as a card (need a repo that belongs to the org)
	// Find a repo owned by the org
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org.ID})

	issueOpt := structs.CreateIssueOption{
		Title: "test issue for org project card",
		Body:  "test issue body",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", org.Name, repo.Name), issueOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var issue structs.Issue
	DecodeJSON(t, resp, &issue)

	// Add the issue as a card to the column
	addCardOpt := structs.AddCardToColumnOption{
		IssueID: issue.ID,
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID), addCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var card structs.ProjectCard
	DecodeJSON(t, resp, &card)
	require.NotNil(t, card.Issue, "card should have an issue")
	assert.Equal(t, issue.ID, card.Issue.ID)

	// Test listing cards in column (should now have one card)
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &cards)
	require.Len(t, cards, 1, "column should have one card")
	require.NotNil(t, cards[0].Issue, "card should have issue reference")
	assert.Equal(t, issue.ID, cards[0].Issue.ID)
	require.NotNil(t, cards[0].Project, "card should have project reference")
	assert.Equal(t, projectID, cards[0].Project.ID)
	require.NotNil(t, cards[0].Column, "card should have column reference")
	assert.Equal(t, columnID, cards[0].Column.ID)

	// Test with invalid column ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/99999/cards", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with invalid project ID
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/99999/columns/%d/cards", org.Name, columnID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIOrganizationProjectCardReordering(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 owns org3

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization, auth_model.AccessTokenScopeWriteIssue)

	// Create a project
	createOpt := structs.CreateProjectOption{
		Title:        "test org project for card reordering",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// Get project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns", org.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.NotEmpty(t, columns, "project should have default columns")

	columnID := columns[0].ID

	// Find a repo owned by the org
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org.ID})

	// Create multiple issues to add as cards
	issues := make([]structs.Issue, 3)
	cards := make([]structs.ProjectCard, 3)

	for i := 0; i < 3; i++ {
		issueOpt := structs.CreateIssueOption{
			Title: fmt.Sprintf("test issue %d for org card reordering", i+1),
			Body:  "test issue body",
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", org.Name, repo.Name), issueOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &issues[i])

		// Add each issue as a card to the column
		addCardOpt := structs.AddCardToColumnOption{
			IssueID: issues[i].ID,
		}

		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID), addCardOpt).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)
		DecodeJSON(t, resp, &cards[i])
	}

	// Get current card order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var originalCards []structs.ProjectCard
	DecodeJSON(t, resp, &originalCards)
	assert.Len(t, originalCards, 3, "column should have three cards")

	// Reorder cards: reverse the order
	reorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{
			{CardID: originalCards[2].ID, Position: 0}, // Move last card to first
			{CardID: originalCards[1].ID, Position: 1}, // Keep middle card in middle
			{CardID: originalCards[0].ID, Position: 2}, // Move first card to last
		},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards/reorder", org.Name, projectID, columnID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify new order
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, columnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var reorderedCards []structs.ProjectCard
	DecodeJSON(t, resp, &reorderedCards)
	assert.Len(t, reorderedCards, 3, "column should still have three cards")

	// Check that the order is reversed
	assert.Equal(t, originalCards[2].ID, reorderedCards[0].ID, "first card should be the original third card")
	assert.Equal(t, originalCards[1].ID, reorderedCards[1].ID, "second card should be the original second card")
	assert.Equal(t, originalCards[0].ID, reorderedCards[2].ID, "third card should be the original first card")

	// Test error cases

	// Test with invalid card ID
	invalidReorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{
			{CardID: 99999, Position: 0},
		},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards/reorder", org.Name, projectID, columnID), invalidReorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with empty card positions
	emptyReorderOpt := structs.ReorderCardsInColumnOption{
		CardPositions: []structs.CardPosition{},
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards/reorder", org.Name, projectID, columnID), emptyReorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusBadRequest)

	// Test with invalid column ID
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/99999/cards/reorder", org.Name, projectID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test with invalid project ID
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/99999/columns/%d/cards/reorder", org.Name, columnID), reorderOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIOrganizationProjectCardMoving(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use org3 which has owner user2
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 owns org3

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization, auth_model.AccessTokenScopeWriteIssue)

	// Create a project
	createOpt := structs.CreateProjectOption{
		Title:        "test org project for card moving",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// Get project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns", org.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.GreaterOrEqual(t, len(columns), 2, "project should have at least 2 columns for moving test")

	sourceColumnID := columns[0].ID
	targetColumnID := columns[1].ID

	// Find a repo owned by the org
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: org.ID})

	// Create an issue to add as a card
	issueOpt := structs.CreateIssueOption{
		Title: "test issue for org card moving",
		Body:  "test issue body",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", org.Name, repo.Name), issueOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var issue structs.Issue
	DecodeJSON(t, resp, &issue)

	// Add the issue as a card to the source column
	addCardOpt := structs.AddCardToColumnOption{
		IssueID: issue.ID,
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, sourceColumnID), addCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var card structs.ProjectCard
	DecodeJSON(t, resp, &card)

	cardID := card.ID

	// Move the card to the target column
	moveCardOpt := structs.MoveProjectCardOption{
		ColumnID: &targetColumnID,
		Position: &[]int64{0}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/cards/%d", org.Name, projectID, cardID), moveCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var movedCard structs.ProjectCard
	DecodeJSON(t, resp, &movedCard)
	require.NotNil(t, movedCard.Column, "moved card should have a column reference")
	assert.Equal(t, targetColumnID, movedCard.Column.ID, "card should be moved to target column")

	// Verify card is in target column
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, targetColumnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var targetCards []structs.ProjectCard
	DecodeJSON(t, resp, &targetCards)
	found := false
	for _, c := range targetCards {
		if c.ID == cardID {
			found = true
			break
		}
	}
	assert.True(t, found, "card should be found in target column")

	// Verify card is no longer in source column
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, sourceColumnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var sourceCards []structs.ProjectCard
	DecodeJSON(t, resp, &sourceCards)
	found = false
	for _, c := range sourceCards {
		if c.ID == cardID {
			found = true
			break
		}
	}
	assert.False(t, found, "card should not be found in source column")

	// Test deleting the card
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/cards/%d", org.Name, projectID, cardID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify card is deleted
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/columns/%d/cards", org.Name, projectID, targetColumnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	DecodeJSON(t, resp, &targetCards)
	found = false
	for _, c := range targetCards {
		if c.ID == cardID {
			found = true
			break
		}
	}
	assert.False(t, found, "card should be deleted from target column")

	// Test error cases

	// Test moving non-existent card
	invalidMoveOpt := structs.MoveProjectCardOption{
		ColumnID: &targetColumnID,
		Position: &[]int64{0}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/cards/99999", org.Name, projectID), invalidMoveOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Test deleting non-existent card
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d/cards/99999", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

func TestAPIRepositoryProjectCardMoving(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Get test data - use repo1 owned by user2
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteIssue)

	// Create a project
	createOpt := structs.CreateProjectOption{
		Title:        "test repo project for card moving",
		Body:         "test project description",
		TemplateType: structs.ProjectTemplateTypeBasicKanban,
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects", owner.Name, repo.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var apiProject structs.Project
	DecodeJSON(t, resp, &apiProject)

	projectID := apiProject.ID

	// Get project columns
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var columns []structs.ProjectColumn
	DecodeJSON(t, resp, &columns)
	assert.GreaterOrEqual(t, len(columns), 2, "project should have at least 2 columns for moving test")

	sourceColumnID := columns[0].ID
	targetColumnID := columns[1].ID

	// Create an issue to add as a card
	issueOpt := structs.CreateIssueOption{
		Title: "test issue for repo card moving",
		Body:  "test issue body",
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/issues", owner.Name, repo.Name), issueOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var issue structs.Issue
	DecodeJSON(t, resp, &issue)

	// Add the issue as a card to the source column
	addCardOpt := structs.AddCardToColumnOption{
		IssueID: issue.ID,
	}

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, sourceColumnID), addCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusCreated)
	var card structs.ProjectCard
	DecodeJSON(t, resp, &card)

	cardID := card.ID

	// Move the card to the target column
	moveCardOpt := structs.MoveProjectCardOption{
		ColumnID: &targetColumnID,
		Position: &[]int64{0}[0],
	}

	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/cards/%d", owner.Name, repo.Name, projectID, cardID), moveCardOpt).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var movedCard structs.ProjectCard
	DecodeJSON(t, resp, &movedCard)
	require.NotNil(t, movedCard.Column, "moved card should have a column reference")
	assert.Equal(t, targetColumnID, movedCard.Column.ID, "card should be moved to target column")

	// Verify card is in target column
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, targetColumnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var targetCards []structs.ProjectCard
	DecodeJSON(t, resp, &targetCards)
	assert.NotEmpty(t, targetCards, "target column should contain the moved card")

	// Find the moved card in the target column
	var foundCard *structs.ProjectCard
	for i := range targetCards {
		if targetCards[i].ID == cardID {
			foundCard = &targetCards[i]
			break
		}
	}
	assert.NotNil(t, foundCard, "moved card should be found in target column")

	// Verify card is no longer in source column
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d/columns/%d/cards", owner.Name, repo.Name, projectID, sourceColumnID)).
		AddTokenAuth(token)
	resp = MakeRequest(t, req, http.StatusOK)
	var sourceCards []structs.ProjectCard
	DecodeJSON(t, resp, &sourceCards)

	// Check that the card is not in the source column anymore
	for _, sourceCard := range sourceCards {
		assert.NotEqual(t, cardID, sourceCard.ID, "moved card should not be in source column")
	}

	// Clean up
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/%s/projects/%d", owner.Name, repo.Name, projectID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}

// TestAPIOrganizationProjectCrossOrgAccess verifies that projects from one org cannot be accessed via another org's endpoint
func TestAPIOrganizationProjectCrossOrgAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Setup: org3 (ID 3) owned by user2
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3, Type: user_model.UserTypeOrganization})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteOrganization)

	// Create a project in org3
	createOpt := structs.CreateProjectOption{
		Title: "cross org security test project",
		Body:  "test",
	}

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/%s/projects", org3.Name), createOpt).
		AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)
	var project structs.Project
	DecodeJSON(t, resp, &project)

	// Verify we can access it via org3's endpoint
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org3.Name, project.ID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusOK)

	// Try to access org3's project via org6's endpoint - should fail
	// Returns 403 (no permission on org6) which also prevents leaking project existence
	org6 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 6, Type: user_model.UserTypeOrganization})
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org6.Name, project.ID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// Try to update org3's project via org6's endpoint - should fail
	updateOpt := structs.EditProjectOption{
		Title: &[]string{"hacked title"}[0],
	}
	req = NewRequestWithJSON(t, "PATCH", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org6.Name, project.ID), updateOpt).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// Try to delete org3's project via org6's endpoint - should fail
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org6.Name, project.ID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)

	// Clean up - delete via correct org endpoint
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/orgs/%s/projects/%d", org3.Name, project.ID)).
		AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)
}
