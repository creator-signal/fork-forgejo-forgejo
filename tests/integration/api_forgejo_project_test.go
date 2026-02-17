// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	v1 "forgejo.org/routers/api/forgejo/v1"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const forgejoAPIBase = "/api/forgejo/v1"

func TestAPIForgejoRepoProjects(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 owns repo1, which has projects enabled and has project ID 1
	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	t.Run("ListProjects", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var projects []v1.Project
		DecodeJSON(t, resp, &projects)
		assert.NotEmpty(t, projects)
		assert.Equal(t, "First project", *projects[0].Title)
		assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))
	})

	t.Run("ListProjects/NoAuth", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Public repo should be accessible without auth
		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects")
		MakeRequest(t, req, http.StatusOK)
	})

	t.Run("CreateProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects", v1.CreateProjectOption{
			Title: "Test Project",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, "Test Project", *project.Title)
		assert.NotNil(t, project.Id)

		// Verify the project appears in the list
		req = NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects?state=all").
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		var projects []v1.Project
		DecodeJSON(t, resp, &projects)

		found := false
		for _, p := range projects {
			if *p.Title == "Test Project" {
				found = true
				break
			}
		}
		assert.True(t, found, "created project should appear in list")
	})

	t.Run("CreateProject/NoAuth", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects", v1.CreateProjectOption{
			Title: "Should Fail",
		})
		MakeRequest(t, req, http.StatusUnauthorized)
	})

	t.Run("CreateProject/EmptyTitle", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects", v1.CreateProjectOption{
			Title: "",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusUnprocessableEntity)
	})

	t.Run("GetProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Project 1 belongs to repo 1
		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/1").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, int64(1), *project.Id)
		assert.Equal(t, "First project", *project.Title)
	})

	t.Run("GetProject/NotFound", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/99999").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("GetProject/WrongRepo", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Project 2 belongs to repo 3, not repo 1
		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/2").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("UpdateProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		newTitle := "Updated Project Title"
		req := NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1", v1.EditProjectOption{
			Title: &newTitle,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, "Updated Project Title", *project.Title)
	})

	t.Run("UpdateProject/Close", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		closedState := v1.EditProjectOptionStateClosed
		req := NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1", v1.EditProjectOption{
			State: &closedState,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, v1.ProjectStateClosed, *project.State)

		// Reopen it
		openState := v1.EditProjectOptionStateOpen
		req = NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1", v1.EditProjectOption{
			State: &openState,
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		DecodeJSON(t, resp, &project)
		assert.Equal(t, v1.ProjectStateOpen, *project.State)
	})
}

func TestAPIForgejoRepoProjectColumns(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	t.Run("ListColumns", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var columns []v1.ProjectColumn
		DecodeJSON(t, resp, &columns)
		assert.Len(t, columns, 3) // To Do, In Progress, Done
		assert.Equal(t, "To Do", *columns[0].Title)
	})

	t.Run("CreateColumn", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns", v1.CreateColumnOption{
			Title: "Review",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var col v1.ProjectColumn
		DecodeJSON(t, resp, &col)
		assert.Equal(t, "Review", *col.Title)
		assert.NotNil(t, col.Id)
	})

	t.Run("UpdateColumn", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		newTitle := "Updated To Do"
		newColor := "#ff0000"
		req := NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns/1", v1.EditColumnOption{
			Title: &newTitle,
			Color: &newColor,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var col v1.ProjectColumn
		DecodeJSON(t, resp, &col)
		assert.Equal(t, "Updated To Do", *col.Title)
		assert.Equal(t, "#ff0000", *col.Color)
	})

	t.Run("UpdateColumn/WrongProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Column 5 belongs to project 2, not project 1
		newTitle := "Should Fail"
		req := NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns/5", v1.EditColumnOption{
			Title: &newTitle,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("MoveColumns", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// List all columns first (prior tests may have added columns)
		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var columns []v1.ProjectColumn
		DecodeJSON(t, resp, &columns)
		require.NotEmpty(t, columns)

		// Reverse the order of all columns
		positions := make([]v1.ColumnPosition, len(columns))
		for i, col := range columns {
			positions[i] = v1.ColumnPosition{
				ColumnId: *col.Id,
				Position: len(columns) - 1 - i,
			}
		}

		req = NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns/move", v1.MoveColumnsOption{
			Columns: positions,
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})
}

func TestAPIForgejoRepoProjectCards(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	t.Run("ListCards", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Column 1 (To Do) in project 1 has card with issue 1
		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns/1/cards").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var cards []v1.ProjectCard
		DecodeJSON(t, resp, &cards)
		assert.NotEmpty(t, cards)
		assert.NotEmpty(t, resp.Header().Get("X-Total-Count"))

		// Verify issue info is present
		require.NotNil(t, cards[0].Issue)
		assert.Equal(t, int64(1), *cards[0].Issue.Id)
	})

	t.Run("AddCard", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Issue 11 is in repo 1 and not yet in the project
		var position int64 = 10
		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns/1/cards", v1.AddCardOption{
			IssueId:  11,
			Position: &position,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var card v1.ProjectCard
		DecodeJSON(t, resp, &card)
		assert.NotNil(t, card.Id)
		require.NotNil(t, card.Issue)
		assert.Equal(t, int64(11), *card.Issue.Id)
	})

	t.Run("MoveCard", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Move card 1 (issue 1 in column 1) to column 2
		var newCol int64 = 2
		req := NewRequestWithJSON(t, "PATCH", forgejoAPIBase+"/repos/user2/repo1/projects/1/cards/1", v1.MoveCardOption{
			ColumnId: &newCol,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var card v1.ProjectCard
		DecodeJSON(t, resp, &card)
		assert.NotNil(t, card.Column)
		assert.Equal(t, int64(2), *card.Column.Id)
	})

	t.Run("DeleteCard", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Delete card 3 (issue 3 in project 1)
		req := NewRequest(t, "DELETE", forgejoAPIBase+"/repos/user2/repo1/projects/1/cards/3").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})
}

func TestAPIForgejoOrgProjects(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 is a member of org3
	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue, auth_model.AccessTokenScopeWriteOrganization)

	var createdProjectID int64

	t.Run("CreateOrgProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		templateType := 1 // basic kanban
		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/orgs/org3/projects", v1.CreateProjectOption{
			Title:        "Org Test Project",
			TemplateType: &templateType,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, "Org Test Project", *project.Title)
		assert.NotNil(t, project.Id)
		createdProjectID = *project.Id
	})

	t.Run("ListOrgProjects", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", forgejoAPIBase+"/orgs/org3/projects").
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var projects []v1.Project
		DecodeJSON(t, resp, &projects)
		assert.NotEmpty(t, projects)
	})

	t.Run("GetOrgProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		require.NotZero(t, createdProjectID, "need created project")

		req := NewRequest(t, "GET", fmt.Sprintf("%s/orgs/org3/projects/%d", forgejoAPIBase, createdProjectID)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, "Org Test Project", *project.Title)
	})

	t.Run("UpdateOrgProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		require.NotZero(t, createdProjectID, "need created project")

		newTitle := "Updated Org Project"
		req := NewRequestWithJSON(t, "PATCH", fmt.Sprintf("%s/orgs/org3/projects/%d", forgejoAPIBase, createdProjectID), v1.EditProjectOption{
			Title: &newTitle,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var project v1.Project
		DecodeJSON(t, resp, &project)
		assert.Equal(t, "Updated Org Project", *project.Title)
	})

	t.Run("OrgProjectColumns", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		require.NotZero(t, createdProjectID, "need created project")

		// List columns (new project should have default columns)
		req := NewRequest(t, "GET", fmt.Sprintf("%s/orgs/org3/projects/%d/columns", forgejoAPIBase, createdProjectID)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var columns []v1.ProjectColumn
		DecodeJSON(t, resp, &columns)
		assert.NotEmpty(t, columns, "new project should have default columns from template")

		// Create a column
		req = NewRequestWithJSON(t, "POST", fmt.Sprintf("%s/orgs/org3/projects/%d/columns", forgejoAPIBase, createdProjectID), v1.CreateColumnOption{
			Title: "Org Review",
		}).AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusCreated)

		var col v1.ProjectColumn
		DecodeJSON(t, resp, &col)
		assert.Equal(t, "Org Review", *col.Title)
	})

	t.Run("DeleteOrgProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		require.NotZero(t, createdProjectID, "need created project")

		req := NewRequest(t, "DELETE", fmt.Sprintf("%s/orgs/org3/projects/%d", forgejoAPIBase, createdProjectID)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)

		// Verify it's gone
		req = NewRequest(t, "GET", fmt.Sprintf("%s/orgs/org3/projects/%d", forgejoAPIBase, createdProjectID)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})
}

func TestAPIForgejoProjectPermissions(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("NonMemberCannotCreateOrgProject", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// user5 is not a member of org3
		session := loginUser(t, "user5")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue, auth_model.AccessTokenScopeWriteOrganization)

		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/orgs/org3/projects", v1.CreateProjectOption{
			Title: "Should Fail",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	})

	t.Run("NonExistentRepo", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

		req := NewRequest(t, "GET", forgejoAPIBase+"/repos/user2/nonexistent/projects").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("NonExistentOrg", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, "user2")
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

		req := NewRequest(t, "GET", forgejoAPIBase+"/orgs/nonexistent/projects").
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})
}

func TestAPIForgejoProjectDeleteColumn(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteIssue)

	t.Run("DeleteColumn", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// First create a column, then delete it
		req := NewRequestWithJSON(t, "POST", forgejoAPIBase+"/repos/user2/repo1/projects/1/columns", v1.CreateColumnOption{
			Title: "Temporary Column",
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusCreated)

		var col v1.ProjectColumn
		DecodeJSON(t, resp, &col)

		// Delete it
		req = NewRequest(t, "DELETE", fmt.Sprintf("%s/repos/user2/repo1/projects/1/columns/%d", forgejoAPIBase, *col.Id)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})
}
