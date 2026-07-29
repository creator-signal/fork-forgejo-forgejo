// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"testing"

	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	project_module "forgejo.org/modules/project"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testPaginationCreateProject creates a project.
func testPaginationCreateProject(t *testing.T, token, userName, repoName, projectName string) api.Project {
	var project api.Project
	jsonRequestWithAuthChecked(t, token, "POST",
		fmt.Sprintf(
			"/api/v1/repos/%v/%v/projects",
			userName,
			repoName,
		),
		&api.CreateProjectOptions{
			Title:        projectName,
			Description:  projectName,
			TemplateType: project_module.APITemplateTypeNone.String(),
			CardType:     project_module.APICardTypeTextOnly.String(),
		},
		http.StatusCreated,
		&project,
	)
	return project
}

// testPaginationCreateColumn creates a column.
func testPaginationCreateColumn(t *testing.T, token, userName, repoName string, projectID int64, columnName string) api.ProjectColumn {
	var projectColumn api.ProjectColumn
	jsonRequestWithAuthChecked(t, token, "POST",
		fmt.Sprintf(
			"/api/v1/repos/%v/%v/projects/%d/columns",
			userName,
			repoName,
			projectID,
		),
		api.CreateProjectColumnOptions{
			Title: columnName,
		},
		http.StatusCreated,
		&projectColumn,
	)
	return projectColumn
}

// testPaginationCreateIssue creates an issue.
func testPaginationCreateIssue(t *testing.T, token, userName, repoName string, projectID, columnID int64, issueName string) api.ProjectIssue {
	// create issue
	var issue api.Issue
	jsonRequestWithAuthChecked(t, token, "POST",
		fmt.Sprintf(
			"/api/v1/repos/%s/%s/issues?state=all",
			userName,
			repoName,
		),
		&api.CreateIssueOption{
			Body:  issueName,
			Title: issueName,
		},
		http.StatusCreated,
		&issue,
	)

	// create project issue
	var projectIssue api.ProjectIssue
	jsonRequestWithAuthChecked(t, token, "POST",
		fmt.Sprintf(
			"/api/v1/repos/%v/%v/projects/%d/columns/%d/issues",
			userName,
			repoName,
			projectID,
			columnID,
		),
		&api.CreateProjectIssueOptions{
			IssueID: issue.ID,
		},
		http.StatusCreated,
		&projectIssue,
	)
	return projectIssue
}

// TestProjectAPIListProjectsPagination tests ListProjects in the Project API
// with pagination.
func TestProjectAPIListProjectsPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	err := unittest.PrepareTestDatabase()
	require.NoError(t, err)

	// user and token, repo
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteProject)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// create projects
	numProjects := 100
	projects := []api.Project{}
	for i := range numProjects {
		n := fmt.Sprintf("project-%d", i)
		projects = append(projects,
			testPaginationCreateProject(t, token, user.LowerName, repo.LowerName, n))
	}

	// list projects
	limit := 10   // maximum number of entries in api call response
	numCalls := 0 // number of performed api calls
	gotProjects := []api.Project{}
	for i := range numProjects {
		var projResp []api.Project
		resp := requestWithAuthChecked(
			t, token, "GET",
			fmt.Sprintf(
				"/api/v1/repos/%v/%v/projects?page=%d&limit=%d",
				user.Name,
				repo.Name,
				i+1,
				limit,
			),
			http.StatusOK,
			&projResp,
		)
		numCalls++

		assert.Equal(t, strconv.Itoa(numProjects), resp.Result().Header.Get("X-Total-Count"))
		assert.NotEmpty(t, resp.Result().Header.Get("Link"))
		gotProjects = append(gotProjects, projResp...)

		if len(gotProjects) == numProjects {
			break
		}
	}
	assert.Len(t, gotProjects, numProjects)
	assert.Equal(t, projects, gotProjects)
	assert.Equal(t, numProjects/limit, numCalls)
}

// TestProjectAPIListProjectColumnsPagination tests ListProjectColumns in the
// Project API with pagination.
func TestProjectAPIListProjectColumnsPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	err := unittest.PrepareTestDatabase()
	require.NoError(t, err)

	// user and token, repo
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteProject)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// create project
	project := testPaginationCreateProject(t, token, user.LowerName, repo.LowerName, "test-project")

	// create columns
	numColumns := 20
	columns := []api.ProjectColumn{}
	for i := range numColumns {
		n := fmt.Sprintf("column-%d", i)
		columns = append(columns,
			testPaginationCreateColumn(t, token, user.LowerName, repo.LowerName, project.ID, n))
	}

	// list columns
	limit := 2    // maximum number of entries in api call response
	numCalls := 0 // number of performed api calls
	gotColumns := []api.ProjectColumn{}
	for i := range numColumns {
		var colResp []api.ProjectColumn
		resp := requestWithAuthChecked(
			t, token, "GET",
			fmt.Sprintf(
				"/api/v1/repos/%v/%v/projects/%d/columns?page=%d&limit=%d",
				user.Name,
				repo.Name,
				project.ID,
				i+1,
				limit,
			),
			http.StatusOK,
			&colResp,
		)
		numCalls++

		assert.Equal(t, strconv.Itoa(numColumns), resp.Result().Header.Get("X-Total-Count"))
		assert.NotEmpty(t, resp.Result().Header.Get("Link"))
		gotColumns = append(gotColumns, colResp...)

		if len(gotColumns) == numColumns {
			break
		}
	}
	assert.Len(t, gotColumns, numColumns)
	assert.Equal(t, columns, gotColumns)
	assert.Equal(t, numColumns/limit, numCalls)
}

// TestProjectAPIListProjectIssuesPagination tests ListProjectColumnIssues and
// ListProjectIssues in the Project API with pagination.
func TestProjectAPIListProjectIssuesPagination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	err := unittest.PrepareTestDatabase()
	require.NoError(t, err)

	// user and token, repo
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user.Name)
	token := getTokenForLoggedInUser(t, session,
		auth_model.AccessTokenScopeWriteProject,
		auth_model.AccessTokenScopeWriteIssue,
	)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// create project
	project := testPaginationCreateProject(t, token, user.LowerName, repo.LowerName, "test-project")

	// create column
	column := testPaginationCreateColumn(t, token, user.LowerName, repo.LowerName, project.ID, "test-column")

	// create issues
	numIssues := 100
	issues := []api.ProjectIssue{}
	for i := range numIssues {
		n := fmt.Sprintf("issue-%d", i)
		issues = append(issues,
			testPaginationCreateIssue(t, token, user.LowerName, repo.LowerName, project.ID, column.ID, n))
	}

	// list issues
	listIssues := func(t *testing.T, url string) {
		limit := 10   // maximum number of entries in api call response
		numCalls := 0 // number of performed api calls
		gotIssues := []api.ProjectIssue{}
		for i := range numIssues {
			var issueResp []api.ProjectIssue
			resp := requestWithAuthChecked(
				t, token, "GET",
				fmt.Sprintf(
					"%s?page=%d&limit=%d",
					url,
					i+1,
					limit,
				),
				http.StatusOK,
				&issueResp,
			)
			numCalls++

			assert.Equal(t, strconv.Itoa(numIssues), resp.Result().Header.Get("X-Total-Count"))
			assert.NotEmpty(t, resp.Result().Header.Get("Link"))
			gotIssues = append(gotIssues, issueResp...)

			if len(gotIssues) == numIssues {
				break
			}
		}
		assert.Len(t, gotIssues, numIssues)
		assert.Equal(t, issues, gotIssues)
		assert.Equal(t, numIssues/limit, numCalls)
	}
	t.Run("ListProjectColumnIssues", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		url := fmt.Sprintf(
			"/api/v1/repos/%v/%v/projects/%d/columns/%d/issues",
			user.Name,
			repo.Name,
			project.ID,
			column.ID,
		)
		listIssues(t, url)
	})
	t.Run("ListProjectIssues", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		url := fmt.Sprintf(
			"/api/v1/repos/%s/%s/projects/%d/issues",
			user.Name,
			repo.Name,
			project.ID,
		)
		listIssues(t, url)
	})
}

// Test the use cases
func TestProjectAPICRUD(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	err := unittest.PrepareTestDatabase()
	require.NoError(t, err)

	// Get repo
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	// Get issue1 and issue2
	issue1 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1})
	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 5}) // issue 2 is PR, issue 5 is no PR but closed

	// User and auth
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)
	writeToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteProject)
	readToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadProject)

	isClosed := func(s string) bool {
		if s == "closed" {
			return true
		}
		return false
	}

	// UC07, UC03: Create, Get project for an owner
	projectOpts := api.CreateProjectOptions{
		Title:        "Project 1",
		Description:  "Test",
		TemplateType: project_module.APITemplateTypeNone.String(),
		CardType:     project_module.APICardTypeTextOnly.String(),
	}

	userPostEndpoint := fmt.Sprintf("/api/v1/users/%v", user2.Name)
	resp := createProject(t, writeToken, userPostEndpoint, &projectOpts)
	var project api.Project
	DecodeJSON(t, resp, &project)

	assert.NotZero(t, project.ID)
	assert.Equal(t, projectOpts.Title, project.Title)
	assert.Equal(t, projectOpts.Description, project.Description)
	assert.Equal(t, user2.Name, project.OwnerName)
	assert.Empty(t, project.RepoName)
	assert.Equal(t, "open", project.Status)
	assert.Equal(t, project_module.APITemplateTypeNone.String(), project.TemplateType)
	assert.Equal(t, project_module.APICardTypeTextOnly.String(), project.CardType)

	userGetEndpoint := fmt.Sprintf("/api/v1/users/%v", user2.Name)
	resp = getProject(t, readToken, userGetEndpoint, project.ID)
	assert.Equal(t, http.StatusOK, resp.Code)
	var projResp api.Project
	DecodeJSON(t, resp, &projResp)

	assert.Equal(t, project.ID, projResp.ID)

	// UC08: Create project for a repository
	t.Run("UC08, UC04: Create, Get project for repository", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// repo1 is owned by user2
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

		projOpts := api.CreateProjectOptions{
			Title:        "Project 2",
			Description:  "Test 2",
			TemplateType: project_module.APITemplateTypeNone.String(),
			CardType:     project_module.APICardTypeTextOnly.String(),
		}

		repoPostEndpoint := fmt.Sprintf("/api/v1/repos/%v/%v", user2.LowerName, repo1.LowerName)
		resp := createProject(t, writeToken, repoPostEndpoint, &projOpts)
		var projResp api.Project
		DecodeJSON(t, resp, &projResp)

		assert.NotZero(t, projResp.ID)
		assert.NotEqual(t, project.ID, projResp.ID)
		assert.Equal(t, projOpts.Title, projResp.Title)
		assert.Equal(t, projOpts.Description, projResp.Description)
		assert.Equal(t, user2.Name, projResp.OwnerName)
		assert.Equal(t, repo.Name, projResp.RepoName)
		assert.Equal(t, "open", projResp.Status)
		assert.Equal(t, project_module.APITemplateTypeNone.String(), projResp.TemplateType)
		assert.Equal(t, project_module.APICardTypeTextOnly.String(), projResp.CardType)

		repoGetEndpoint := fmt.Sprintf("/api/v1/repos/%v/%v", user2.Name, repo1.LowerName)
		resp = getProject(t, readToken, repoGetEndpoint, projResp.ID)
		assert.Equal(t, http.StatusOK, resp.Code)

		var projResp2 api.Project
		DecodeJSON(t, resp, &projResp2)

		assert.Equal(t, projResp.ID, projResp2.ID)
		assert.Equal(t, projResp, projResp2)
	})

	// UC09: Create columns in a project
	createPCOpt1 := api.CreateProjectColumnOptions{
		Title:   "Col1",
		Default: true, // TODO: move check if first column to create column and remove this again?
	}

	endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns", user2.Name, project.ID)
	resp = jsonRequestWithAuth(t, writeToken, "POST", endpoint, createPCOpt1)
	assert.Equal(t, http.StatusCreated, resp.Code)
	var projectColumn1 api.ProjectColumn
	DecodeJSON(t, resp, &projectColumn1) // First column is always default column

	// Color can be nil
	assert.NotZero(t, projectColumn1.ID)
	assert.Equal(t, createPCOpt1.Title, projectColumn1.Title)
	assert.Equal(t, project.ID, projectColumn1.ProjectID)
	assert.True(t, projectColumn1.Default)
	assert.NotNil(t, projectColumn1.Sorting) // Sorting is zero by default, but "NOT NULL" according to DB model

	createPCOpt2 := api.CreateProjectColumnOptions{
		Title: "Col2",
	}

	resp = jsonRequestWithAuth(t, writeToken, "POST", endpoint, createPCOpt2)
	assert.Equal(t, http.StatusCreated, resp.Code)

	var projectColumn2 api.ProjectColumn
	DecodeJSON(t, resp, &projectColumn2)

	assert.NotZero(t, projectColumn2.ID)
	assert.NotEqual(t, projectColumn1.ID, projectColumn2.ID)
	assert.Equal(t, createPCOpt2.Title, projectColumn2.Title)
	assert.Equal(t, project.ID, projectColumn2.ProjectID)
	assert.False(t, projectColumn2.Default)
	assert.NotEqual(t, projectColumn1.Sorting, projectColumn2.Sorting)

	// UC10: Add issue to a project, to the default column
	createPIOpt1 := api.CreateProjectIssueOptions{
		IssueID: issue1.ID,
	}

	endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/issues", user2.Name, project.ID)
	resp = jsonRequestWithAuth(t, writeToken, "POST", endpoint, createPIOpt1)
	assert.Equal(t, http.StatusCreated, resp.Code)

	var projectIssue1 api.ProjectIssue
	DecodeJSON(t, resp, &projectIssue1)

	assert.NotZero(t, projectIssue1.ID)
	assert.Equal(t, project.ID, projectIssue1.ProjectID)
	assert.Equal(t, projectColumn1.ID, projectIssue1.ProjectColumnID)
	assert.NotNil(t, projectIssue1.Sorting) // Sorting is zero by default, but "NOT NULL" according to DB model

	// UC11: Add issue directly to a column of a project
	createPIOpt2 := api.CreateProjectIssueOptions{
		IssueID: issue2.ID,
	}

	endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues", user2.Name, project.ID, projectColumn1.ID)
	resp = jsonRequestWithAuth(t, writeToken, "POST", endpoint, createPIOpt2)
	assert.Equal(t, http.StatusCreated, resp.Code)

	var projectIssue2 api.ProjectIssue
	DecodeJSON(t, resp, &projectIssue2)

	assert.NotZero(t, projectIssue2.ID)
	assert.NotEqual(t, projectIssue1.ID, projectIssue2.ID)
	assert.Equal(t, project.ID, projectIssue2.ProjectID)
	assert.Equal(t, projectColumn1.ID, projectIssue2.ProjectColumnID)
	assert.NotEqual(t, projectIssue1.Sorting, projectIssue2.Sorting)

	// UC18: Update properties of a project
	t.Run("UC18: Update properties of project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		projOpts := api.CreateProjectOptions{
			Title:       "Project 15",
			Description: "ABC",
			CardType:    project_module.APICardTypeImagesAndText.String(),
			Status:      project.Status,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v", user2.Name, project.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, projOpts)
		assert.Equal(t, http.StatusOK, resp.Code)
		p := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: project.ID})

		assert.Equal(t, projOpts.Title, p.Title)
		assert.Equal(t, projOpts.Description, p.Description)
		assert.Equal(t, projOpts.CardType, p.CardType.ToAPICardType().String())
		assert.Equal(t, isClosed(projOpts.Status), p.IsClosed)
	})

	// UC21: Change status of a project (open, closed)
	t.Run("UC21: Change status of project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// close project
		projOpts := api.CreateProjectOptions{
			Title:       "Project 15",
			Description: "ABC",
			CardType:    project_module.APICardTypeImagesAndText.String(),
			Status:      project_module.APIStatusClosed.String(),
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v", user2.Name, project.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, projOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		p := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: project.ID})
		assert.Equal(t, isClosed(projOpts.Status), p.IsClosed)

		// re-open project
		projOpts = api.CreateProjectOptions{
			Title:       "Project 15",
			Description: "ABC",
			CardType:    project_module.APICardTypeImagesAndText.String(),
			Status:      project_module.APIStatusOpen.String(),
		}
		jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, projOpts)

		p = unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: project.ID})
		assert.Equal(t, isClosed(projOpts.Status), p.IsClosed)
	})

	// UC19: Update properties of a column
	t.Run("UC19: Update properties of column", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		colOpts := api.CreateProjectColumnOptions{
			Color:   "#00aabb",
			Title:   "Backlog",
			Default: projectColumn1.Default,
			Sorting: projectColumn1.Sorting,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn1.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, colOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		c := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn1.ID, ProjectID: project.ID})

		assert.Equal(t, colOpts.Color, c.Color)
		assert.Equal(t, colOpts.Title, c.Title)
	})

	// UC20: Set default column of a project
	t.Run("UC20: Set default column of project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// set second column as default
		colOpts := api.CreateProjectColumnOptions{
			Default: true,
			Sorting: projectColumn2.Sorting,
			Title:   projectColumn2.Title,
			Color:   projectColumn2.Color,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn2.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, colOpts)
		assert.Equal(t, http.StatusOK, resp.Code)
		c1 := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn1.ID, ProjectID: project.ID})
		c2 := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn2.ID, ProjectID: project.ID})

		assert.False(t, c1.Default)
		assert.True(t, c2.Default)

		// set first column as default again
		colOpts = api.CreateProjectColumnOptions{
			Default: true,
			Sorting: projectColumn1.Sorting,
			Title:   projectColumn1.Title,
			Color:   projectColumn1.Color,
		}
		endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn1.ID)
		resp = jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, colOpts)
		assert.Equal(t, http.StatusOK, resp.Code)
		c1 = unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn1.ID, ProjectID: project.ID})
		c2 = unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn2.ID, ProjectID: project.ID})

		assert.True(t, c1.Default)
		assert.False(t, c2.Default)
	})

	// UC12: Reorder column in a project
	t.Run("UC12: Reorder column in project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// move second column to first column's position
		colOpts := api.CreateProjectColumnOptions{
			Sorting: projectColumn1.Sorting,
			Default: projectColumn2.Default,
			Title:   projectColumn2.Title,
			Color:   projectColumn2.Color,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn2.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, colOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		// query new values explicitly
		c1 := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn1.ID, ProjectID: project.ID})
		c2 := unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn2.ID, ProjectID: project.ID})

		assert.Less(t, c2.Sorting, c1.Sorting)

		// move first column back to first position
		colOpts = api.CreateProjectColumnOptions{
			Sorting: projectColumn1.Sorting,
			Default: projectColumn1.Default,
			Title:   projectColumn1.Title,
			Color:   projectColumn1.Color,
		}
		endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn1.ID)
		resp = jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, colOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		c1 = unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn1.ID, ProjectID: project.ID})
		c2 = unittest.AssertExistsAndLoadBean(t, &project_model.Column{ID: projectColumn2.ID, ProjectID: project.ID})

		assert.Less(t, c1.Sorting, c2.Sorting)
	})

	// UC13: Reorder issue in a column of a project
	t.Run("UC13: Reorder issue in column of project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		assert.Equal(t, projectIssue1.ProjectColumnID, projectIssue2.ProjectColumnID)

		// move second issue to first issue's position
		updatePCIOpts := api.UpdateProjectColumnIssueOptions{
			ProjectColumnID: projectIssue2.ProjectColumnID,
			Sorting:         projectIssue1.Sorting,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues/%v", user2.Name, project.ID, projectIssue2.ProjectColumnID, projectIssue2.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, updatePCIOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		i1 := unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue1.ID, ProjectID: project.ID})
		i2 := unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue2.ID, ProjectID: project.ID})

		assert.Less(t, i2.Sorting, i1.Sorting)

		// move first issue back to first position
		updatePCIOpts = api.UpdateProjectColumnIssueOptions{
			ProjectColumnID: projectColumn1.ID,
			Sorting:         projectIssue1.Sorting,
		}
		endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues/%v", user2.Name, project.ID, projectIssue1.ProjectColumnID, projectIssue1.ID)
		resp = jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, updatePCIOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		i1 = unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue1.ID, ProjectID: project.ID})
		i2 = unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue2.ID, ProjectID: project.ID})

		assert.Less(t, i1.Sorting, i2.Sorting)
	})

	// UC14: Move issue from one column to a different column of the same project
	t.Run("UC14: Move issue to other column of project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// move second issue to second column
		updatePCIOpts := api.UpdateProjectColumnIssueOptions{
			ProjectColumnID: projectColumn2.ID,
		}
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues/%v", user2.Name, project.ID, projectColumn1.ID, projectIssue2.ID)
		resp := jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, updatePCIOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		i2 := unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue2.ID, ProjectID: project.ID})

		assert.Equal(t, updatePCIOpts.ProjectColumnID, i2.ProjectColumnID)

		// move second issue back to first column
		updatePCIOpts = api.UpdateProjectColumnIssueOptions{
			ProjectColumnID: projectColumn1.ID,
		}
		endpoint = fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues/%v", user2.Name, project.ID, projectColumn2.ID, projectIssue2.ID)
		resp = jsonRequestWithAuth(t, writeToken, "PATCH", endpoint, updatePCIOpts)
		assert.Equal(t, http.StatusOK, resp.Code)

		i2 = unittest.AssertExistsAndLoadBean(t, &project_model.ProjectIssue{ID: projectIssue2.ID, ProjectID: project.ID})

		assert.Equal(t, updatePCIOpts.ProjectColumnID, i2.ProjectColumnID)
	})

	// UC01: List projects of an owner (user/organization)
	t.Run("UC01: List projects of owner", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		var projResp []*api.Project
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects", user2.Name)
		resp := requestWithAuth(t, readToken, "GET", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		DecodeJSON(t, resp, &projResp)

		// there are already some projects in the db for user2
		if !slices.ContainsFunc(projResp, func(p *api.Project) bool {
			return p.ID == project.ID
		}) {
			t.Error("project not in project list")
		}
	})

	// UC02: List projects of a repository
	t.Run("UC02: List projects of repository", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		var projResp []api.Project
		endpoint := fmt.Sprintf("/api/v1/repos/%v/%v/projects", user2.Name, repo.Name)
		resp := requestWithAuth(t, readToken, "GET", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		DecodeJSON(t, resp, &projResp)

		assert.NotEmpty(t, projResp)
	})

	// UC05: List colums of a project
	t.Run("UC05: List columns of a project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		var colResp []api.ProjectColumn
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns", user2.Name, project.ID)
		resp := requestWithAuth(t, readToken, "GET", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		DecodeJSON(t, resp, &colResp)

		// We make sure the order is fixed
		assert.Equal(t, projectColumn1.ID, colResp[0].ID)
		assert.Equal(t, projectColumn2.ID, colResp[1].ID)
	})

	// UC06: List issues in a column of a project
	t.Run("UC06: List issues in a column of a project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		var issueResp []*api.ProjectIssue
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues", user2.Name, project.ID, projectColumn1.ID)
		resp := requestWithAuth(t, readToken, "GET", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		DecodeJSON(t, resp, &issueResp)

		if !slices.ContainsFunc(issueResp, func(p *api.ProjectIssue) bool {
			return p.ID == projectIssue1.ID
		}) {
			t.Error("project issue not in project issue list")
		}
	})

	// UC22: List issues in a project
	t.Run("UC22: List issues in a a project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		var issueResp []*api.ProjectIssue
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/issues", user2.Name, project.ID)
		resp := requestWithAuth(t, readToken, "GET", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)
		DecodeJSON(t, resp, &issueResp)

		if !slices.ContainsFunc(issueResp, func(p *api.ProjectIssue) bool {
			return p.ID == projectIssue1.ID
		}) {
			t.Error("project issue not in project issue list")
		}
	})

	// UC15: Remove issue from (a column of) a project
	t.Run("UC15: Remove issue from a column of a project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v/issues/%v", user2.Name, project.ID, projectColumn1.ID, projectIssue1.ID)
		resp := requestWithAuth(t, writeToken, "DELETE", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)

		unittest.AssertNotExistsBean(t, &project_model.ProjectIssue{
			ID:        projectIssue1.ID,
			IssueID:   projectIssue1.IssueID,
			ProjectID: projectIssue1.ProjectID,
		})
	})

	// UC16: Remove column from a project
	t.Run("UC16: Remove column from a project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		// cannot delete default column -> delete column 2
		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v/columns/%v", user2.Name, project.ID, projectColumn2.ID)
		resp := requestWithAuth(t, writeToken, "DELETE", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)

		unittest.AssertNotExistsBean(t, &project_model.Column{
			ID:        projectColumn2.ID,
			ProjectID: projectColumn2.ProjectID,
		})
	})

	// UC17: Remove project
	t.Run("UC17: Remove project", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		endpoint := fmt.Sprintf("/api/v1/users/%v/projects/%v", user2.Name, project.ID)
		resp := requestWithAuth(t, writeToken, "DELETE", endpoint)
		assert.Equal(t, http.StatusOK, resp.Code)

		unittest.AssertNotExistsBean(t, &project_model.Project{
			ID: project.ID,
		})
	})
}

func createOrg(t *testing.T, token string, opts *api.CreateOrgOption) *httptest.ResponseRecorder {
	endpoint := "/api/v1/orgs"
	return jsonRequestWithAuth(t, token, "POST", endpoint, opts)
}

func createTeamForOrg(t *testing.T, token, orgName string, opts *api.CreateTeamOption) *httptest.ResponseRecorder {
	endpoint := fmt.Sprintf("/api/v1/orgs/%v/teams", orgName)
	return jsonRequestWithAuth(t, token, "POST", endpoint, opts)
}

func addOrRemoveTeamUser(t *testing.T, token, userName, method string, teamID int64) *httptest.ResponseRecorder {
	endpoint := fmt.Sprintf("/api/v1//teams/%v/members/%v", teamID, userName)
	return requestWithAuth(t, token, method, endpoint)
}

func addorRemoveCollaboratorToRepo(t *testing.T, token, owner, repoName, user, method string, opts *api.AddCollaboratorOption) *httptest.ResponseRecorder {
	endpoint := fmt.Sprintf("/api/v1/repos/%v/%v/collaborators/%v", owner, repoName, user)
	return jsonRequestWithAuth(t, token, method, endpoint, opts)
}

func createUserRepo(t *testing.T, token string, opts *api.CreateRepoOption) *httptest.ResponseRecorder {
	endpoint := "/api/v1/user/repos"
	return jsonRequestWithAuth(t, token, "POST", endpoint, opts)
}

func createProject(t *testing.T, token, projectAPIBaseString string, opts *api.CreateProjectOptions) *httptest.ResponseRecorder {
	projectAPIString := projectAPIBaseString + "/projects"
	return jsonRequestWithAuth(t, token, "POST", projectAPIString, opts)
}

func getProject(t *testing.T, token, projectAPIBaseString string, pID int64) *httptest.ResponseRecorder {
	return projectsIDEndpoint(t, token, "GET", projectAPIBaseString, pID)
}

func deleteProject(t *testing.T, token, projectAPIBaseString string, pID int64) *httptest.ResponseRecorder {
	return projectsIDEndpoint(t, token, "DELETE", projectAPIBaseString, pID)
}

func projectsIDEndpoint(t *testing.T, token, method, projectAPIBaseString string, pID int64) *httptest.ResponseRecorder {
	projectAPIString := fmt.Sprintf("%v/projects/%v", projectAPIBaseString, pID)
	return requestWithAuth(t, token, method, projectAPIString)
}

func requestWithAuth(t *testing.T, token, method, endpoint string) *httptest.ResponseRecorder {
	req := NewRequest(
		t, method,
		endpoint,
	).AddTokenAuth(token)
	resp := MakeRequest(t, req, -1) // We don't want an error here. Instead we'll look at the response later
	return resp
}

func requestWithAuthChecked[T any](
	t *testing.T,
	token, method, endpoint string,
	status int,
	retval *T,
) *httptest.ResponseRecorder {
	resp := requestWithAuth(t, token, method, endpoint)
	require.Equal(t, status, resp.Code)
	DecodeJSON(t, resp, retval)
	return resp
}

func jsonRequestWithAuth(t *testing.T, token, method, endpoint string, opts any) *httptest.ResponseRecorder {
	req := NewRequestWithJSON(
		t, method,
		endpoint,
		&opts,
	).AddTokenAuth(token)
	resp := MakeRequest(t, req, -1)
	return resp
}

func jsonRequestWithAuthChecked[T any](
	t *testing.T,
	token, method, endpoint string,
	opts any,
	status int,
	retval *T,
) {
	resp := jsonRequestWithAuth(t, token, method, endpoint, opts)
	require.Equal(t, status, resp.Code)
	DecodeJSON(t, resp, retval)
}

type runProjectActionsOpts struct {
	token         string
	ownerName     string
	repoName      string
	projectType   project_module.APIOwnerType
	projectID     int64
	projectOpts   *api.CreateProjectOptions
	shouldSucceed bool
}

func getProjectAPIBaseString(opts *runProjectActionsOpts) string {
	var projectAPIBaseString string
	switch opts.projectType {
	case project_module.APIOwnerTypeIndividual:
		projectAPIBaseString = "/api/v1/users/" + opts.ownerName
	case project_module.APIOwnerTypeOrganization:
		projectAPIBaseString = "/api/v1/orgs/" + opts.ownerName
	case project_module.APIOwnerTypeRepository:
		projectAPIBaseString = "/api/v1/repos/" + opts.ownerName + "/" + opts.repoName
	}
	return projectAPIBaseString
}

func runProjectWriteActions(t *testing.T, opts *runProjectActionsOpts) {
	projectAPIBaseString := getProjectAPIBaseString(opts)
	// Create Project
	resp := createProject(t, opts.token, projectAPIBaseString, opts.projectOpts)
	var proj *api.Project
	if opts.shouldSucceed {
		require.Equal(t, http.StatusCreated, resp.Code)
		DecodeJSON(t, resp, &proj)
		assert.Equal(t, opts.projectOpts.Title, proj.Title)
		// Delete Project
		resp = deleteProject(t, opts.token, projectAPIBaseString, proj.ID)
		assert.Equal(t, http.StatusOK, resp.Code)
	} else {
		assert.NotEqual(t, http.StatusCreated, resp.Code)
	}
}

func runProjectReadActions(t *testing.T, opts *runProjectActionsOpts) {
	projectAPIBaseString := getProjectAPIBaseString(opts)
	// Get Project
	resp := getProject(t, opts.token, projectAPIBaseString, opts.projectID)
	if opts.shouldSucceed {
		assert.Equal(t, http.StatusOK, resp.Code)
	} else {
		assert.NotEqual(t, http.StatusOK, resp.Code)
	}
}

func TestProjectAPIPermissionHandling(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	err := unittest.PrepareTestDatabase()
	require.NoError(t, err)

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	session := loginUser(t, user2.Name)
	userWriteToken := getTokenForLoggedInUser(t,
		session,
		auth_model.AccessTokenScopeWriteProject,
		auth_model.AccessTokenScopeWriteOrganization,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
		auth_model.AccessTokenScopeWriteIssue,
	)
	userReadToken := getTokenForLoggedInUser(t,
		session,
		auth_model.AccessTokenScopeReadProject,
		auth_model.AccessTokenScopeReadOrganization,
		auth_model.AccessTokenScopeReadRepository,
		auth_model.AccessTokenScopeReadUser,
		auth_model.AccessTokenScopeReadIssue,
	)

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	session = loginUser(t, user1.Name)
	adminWriteToken := getTokenForLoggedInUser(t,
		session,
		auth_model.AccessTokenScopeWriteProject,
		auth_model.AccessTokenScopeWriteOrganization,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
		auth_model.AccessTokenScopeWriteIssue,
	)

	projectOpts := &api.CreateProjectOptions{
		Title:        "Test Project",
		Description:  "Test Project",
		TemplateType: "none",
		CardType:     "text_only",
		Status:       "open",
	}
	runOpts := &runProjectActionsOpts{
		token:         userWriteToken,
		projectOpts:   projectOpts,
		shouldSucceed: true,
	}

	// Case: Public Org where User2 is owner
	pubOrgOpts := &api.CreateOrgOption{
		UserName:   "pubUser2Org",
		Email:      "test@example.com",
		Visibility: "public",
	}
	resp := createOrg(t, userWriteToken, pubOrgOpts)
	var pubUser2Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &pubUser2Org)

	// Run actions
	runOpts.ownerName = pubUser2Org.Name
	runOpts.projectType = project_module.APIOwnerTypeOrganization
	runProjectWriteActions(t, runOpts)

	// Case: Limited Org where User2 is owner
	limOrgOpts := &api.CreateOrgOption{
		UserName:   "limUser2Org",
		Email:      "test@example.com",
		Visibility: "limited",
	}
	resp = createOrg(t, userWriteToken, limOrgOpts)
	var limUser2Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &limUser2Org)

	// Run actions
	runOpts.ownerName = limUser2Org.Name
	runProjectWriteActions(t, runOpts)

	// Case: Private Org where User2 is owner
	privOrgOpts := &api.CreateOrgOption{
		UserName:   "privUser2Org",
		Email:      "test@example.com",
		Visibility: "limited",
	}
	resp = createOrg(t, userWriteToken, privOrgOpts)
	var privUser2Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &privUser2Org)

	// Run actions
	runOpts.ownerName = privUser2Org.Name
	runProjectWriteActions(t, runOpts)

	// Case: Repo where User2 is owner
	repoOpts := &api.CreateRepoOption{
		Name: "user2Repo",
	}
	resp = createUserRepo(t, userWriteToken, repoOpts)
	var user2Repo *api.Repository
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &user2Repo)

	// Run actions
	runOpts.ownerName = user2.Name
	runOpts.repoName = user2Repo.Name
	runOpts.projectType = project_module.APIOwnerTypeRepository
	runProjectWriteActions(t, runOpts)

	// Case: Project where User2 is owner
	// Run actions
	runOpts.ownerName = user2.Name
	runOpts.repoName = ""
	runOpts.projectType = project_module.APIOwnerTypeIndividual
	runProjectWriteActions(t, runOpts)

	// Case: Public Org where User2 team member with write access
	pubOrgOpts = &api.CreateOrgOption{
		UserName:   "pubUser1Org",
		Email:      "test@example.com",
		Visibility: "public",
	}
	pubOrgTeamOpts := &api.CreateTeamOption{
		Name:       "CanWriteProjects",
		Permission: "write",
		UnitsMap:   map[string]string{"project": "write"},
	}
	resp = createOrg(t, adminWriteToken, pubOrgOpts)
	var pubUser1Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &pubUser1Org)

	var pubUser1OrgTeam *api.Team
	resp = createTeamForOrg(t, adminWriteToken, pubUser1Org.Name, pubOrgTeamOpts)
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &pubUser1OrgTeam)

	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "PUT", pubUser1OrgTeam.ID)

	runOpts.ownerName = pubUser1Org.Name
	runOpts.projectType = project_module.APIOwnerTypeOrganization
	runOpts.token = adminWriteToken
	runProjectWriteActions(t, runOpts)

	// Case: Limited Org where User2 team member with write access
	limOrgOpts = &api.CreateOrgOption{
		UserName:   "limUser1Org",
		Email:      "test@example.com",
		Visibility: "limited",
	}
	limOrgTeamOpts := &api.CreateTeamOption{
		Name:       "CanWriteProjects",
		Permission: "write",
		UnitsMap:   map[string]string{"project": "write"},
	}
	resp = createOrg(t, adminWriteToken, limOrgOpts)
	var limUser1Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &limUser1Org)

	var limUser1OrgTeam *api.Team
	resp = createTeamForOrg(t, adminWriteToken, limUser1Org.Name, limOrgTeamOpts)
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &limUser1OrgTeam)

	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "PUT", limUser1OrgTeam.ID)

	runOpts.ownerName = limUser1Org.Name
	runProjectWriteActions(t, runOpts)

	// Case: Private Org where User2 team member with write access
	privOrgOpts = &api.CreateOrgOption{
		UserName:   "privUser1Org",
		Email:      "test@example.com",
		Visibility: "private",
	}
	privOrgTeamOpts := &api.CreateTeamOption{
		Name:       "CanWriteProjects",
		Permission: "write",
		UnitsMap:   map[string]string{"project": "write"},
	}
	resp = createOrg(t, adminWriteToken, privOrgOpts)
	var privUser1Org *api.Organization
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &privUser1Org)

	var privUser1OrgTeam *api.Team
	resp = createTeamForOrg(t, adminWriteToken, privUser1Org.Name, privOrgTeamOpts)
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &privUser1OrgTeam)

	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "PUT", privUser1OrgTeam.ID)

	runOpts.ownerName = privUser1Org.Name
	runProjectWriteActions(t, runOpts)

	// Case: Repo where User2 is not owner, collaborator with write/read access
	repoOpts = &api.CreateRepoOption{
		Name: "user1Repo",
	}
	resp = createUserRepo(t, adminWriteToken, repoOpts)
	var user1Repo *api.Repository
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &user1Repo)

	writePerm := "write"
	readPerm := "read"
	collabOpts := &api.AddCollaboratorOption{
		Permission: &writePerm,
	}
	addorRemoveCollaboratorToRepo(t, adminWriteToken, user1.Name, repoOpts.Name, user2.Name, "PUT", collabOpts)

	// Run actions
	runOpts.ownerName = user1.Name
	runOpts.repoName = repoOpts.Name
	runOpts.projectType = project_module.APIOwnerTypeRepository
	runProjectWriteActions(t, runOpts)

	// Case: Repo where User2 is not owner
	addorRemoveCollaboratorToRepo(t, adminWriteToken, user1.Name, repoOpts.Name, user2.Name, "DELETE", collabOpts)
	runOpts.shouldSucceed = false
	runOpts.token = userWriteToken
	runProjectWriteActions(t, runOpts)

	// Case: Repo where User2 is collaborator with read access
	collabOpts.Permission = &readPerm
	addorRemoveCollaboratorToRepo(t, adminWriteToken, user1.Name, repoOpts.Name, user2.Name, "PUT", collabOpts)
	runProjectWriteActions(t, runOpts)

	// Case: Public Org where User2 is not member
	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "DELETE", pubUser1OrgTeam.ID)
	runOpts.ownerName = pubUser1Org.Name
	runOpts.repoName = ""
	runOpts.projectType = project_module.APIOwnerTypeOrganization
	runProjectWriteActions(t, runOpts)

	// Case: Limited Org where User2 is not member
	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "DELETE", limUser1OrgTeam.ID)
	runOpts.ownerName = limUser1Org.Name
	runProjectWriteActions(t, runOpts)
	runOpts.token = userReadToken
	runProjectReadActions(t, runOpts)

	// Case: Private Org where User2 is not member
	_ = addOrRemoveTeamUser(t, adminWriteToken, user2.Name, "DELETE", privUser1OrgTeam.ID)
	runOpts.ownerName = privUser1Org.Name
	runOpts.token = userWriteToken
	runProjectWriteActions(t, runOpts)
	runOpts.token = userReadToken
	runProjectReadActions(t, runOpts)

	// Case: Project where User2 is not owner - e.g. try deleting other peoples project
	var delProj *api.Project
	baseString := getProjectAPIBaseString(runOpts)
	resp = createProject(t, adminWriteToken, baseString, runOpts.projectOpts)
	require.Equal(t, http.StatusCreated, resp.Code)
	DecodeJSON(t, resp, &delProj)
	resp = deleteProject(t, userWriteToken, baseString, delProj.ID)
	assert.Equal(t, http.StatusForbidden, resp.Code)
}
