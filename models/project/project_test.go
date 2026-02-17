// Copyright 2020 The Gitea Authors. All rights reserved.
// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"fmt"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsProjectTypeValid(t *testing.T) {
	const UnknownType Type = 15

	cases := []struct {
		typ   Type
		valid bool
	}{
		{TypeIndividual, true},
		{TypeRepository, true},
		{TypeOrganization, true},
		{UnknownType, false},
	}

	for _, v := range cases {
		assert.Equal(t, v.valid, IsTypeValid(v.typ))
	}
}

func TestGetProjects(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	projects, err := db.Find[Project](db.DefaultContext, SearchOptions{RepoID: 1})
	require.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)

	projects, err = db.Find[Project](db.DefaultContext, SearchOptions{RepoID: 3})
	require.NoError(t, err)

	// 1 value for this repo exists in the fixtures
	assert.Len(t, projects, 1)
}

func TestChangeProjectStatusClosedDate(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	p := &Project{
		Type:        TypeRepository,
		CardType:    CardTypeTextOnly,
		Title:       "Status Test Project",
		RepoID:      1,
		CreatedUnix: timeutil.TimeStampNow(),
		CreatorID:   2,
	}
	require.NoError(t, NewProject(db.DefaultContext, p))

	// Close it — ClosedDateUnix should be set
	require.NoError(t, ChangeProjectStatus(db.DefaultContext, p, true))
	closed, err := GetProjectByID(db.DefaultContext, p.ID)
	require.NoError(t, err)
	assert.True(t, closed.IsClosed)
	assert.Positive(t, int64(closed.ClosedDateUnix))

	// Reopen — ClosedDateUnix should be cleared
	require.NoError(t, ChangeProjectStatus(db.DefaultContext, closed, false))
	reopened, err := GetProjectByID(db.DefaultContext, p.ID)
	require.NoError(t, err)
	assert.False(t, reopened.IsClosed)
	assert.Equal(t, int64(0), int64(reopened.ClosedDateUnix))
}

func TestProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	project := &Project{
		Type:         TypeRepository,
		TemplateType: TemplateTypeBasicKanban,
		CardType:     CardTypeTextOnly,
		Title:        "New Project",
		RepoID:       1,
		CreatedUnix:  timeutil.TimeStampNow(),
		CreatorID:    2,
	}

	require.NoError(t, NewProject(db.DefaultContext, project))

	_, err := GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	// Update project
	project.Title = "Updated title"
	require.NoError(t, UpdateProject(db.DefaultContext, project))

	projectFromDB, err := GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	assert.Equal(t, project.Title, projectFromDB.Title)

	require.NoError(t, ChangeProjectStatus(db.DefaultContext, project, true))

	// Retrieve from DB afresh to check if it is truly closed
	projectFromDB, err = GetProjectByID(db.DefaultContext, project.ID)
	require.NoError(t, err)

	assert.True(t, projectFromDB.IsClosed)
}

func TestProjectsSort(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	tests := []struct {
		sortType string
		wants    []int64
	}{
		{
			sortType: "default",
			wants:    []int64{1, 3, 2, 7, 6, 5, 4},
		},
		{
			sortType: "oldest",
			wants:    []int64{4, 5, 6, 7, 2, 3, 1},
		},
		{
			sortType: "recentupdate",
			wants:    []int64{1, 3, 2, 7, 6, 5, 4},
		},
		{
			sortType: "leastupdate",
			wants:    []int64{4, 5, 6, 7, 2, 3, 1},
		},
	}

	for _, tt := range tests {
		projects, count, err := db.FindAndCount[Project](db.DefaultContext, SearchOptions{
			OrderBy: GetSearchOrderByBySortType(tt.sortType),
		})
		require.NoError(t, err)
		assert.Equal(t, int64(7), count)
		if assert.Len(t, projects, 7) {
			for i := range projects {
				assert.Equal(t, tt.wants[i], projects[i].ID)
			}
		}
	}
}

func TestGetProjectForUserByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	found := func(t *testing.T, uid, id int64) {
		t.Helper()

		p, err := GetProjectForUserByID(t.Context(), uid, id)
		require.NoError(t, err)
		if assert.NotNil(t, p) {
			assert.Equal(t, id, p.ID)
		}
	}

	notFound := func(t *testing.T, uid, id int64) {
		t.Helper()

		p, err := GetProjectForUserByID(t.Context(), uid, id)
		require.ErrorIs(t, err, ErrProjectNotExist{ID: id})
		assert.Nil(t, p)
	}

	found(t, 2, 4)
	found(t, 2, 5)
	found(t, 2, 6)
	found(t, 3, 7)
	notFound(t, 1, 4)
	notFound(t, 1, 5)
	notFound(t, 1, 6)
	notFound(t, 1, 7)
}

func TestChangeProjectStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Unchanged", func(t *testing.T) {
		project := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})

		require.NoError(t, ChangeProjectStatus(t.Context(), project, project.IsClosed))

		projectAfter := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
		assert.Equal(t, project.IsClosed, projectAfter.IsClosed)
	})

	t.Run("Normal", func(t *testing.T) {
		project := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
		isClosed := !project.IsClosed
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: project.RepoID})

		require.NoError(t, ChangeProjectStatus(t.Context(), project, isClosed))

		projectAfter := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
		repoAfter := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: project.RepoID})
		assert.Equal(t, isClosed, projectAfter.IsClosed)
		assert.Equal(t, repo.NumProjects, repoAfter.NumProjects)
		assert.Equal(t, repo.NumOpenProjects-1, repoAfter.NumOpenProjects)
		assert.Equal(t, repo.NumClosedProjects+1, repoAfter.NumClosedProjects)
	})

	t.Run("Invalid ID", func(t *testing.T) {
		project := &Project{ID: 1001, RepoID: 1}
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: project.RepoID})

		require.NoError(t, ChangeProjectStatus(t.Context(), project, true))

		repoAfter := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: project.RepoID})
		assert.Equal(t, repo.NumProjects, repoAfter.NumProjects)
		assert.Equal(t, repo.NumOpenProjects, repoAfter.NumOpenProjects)
		assert.Equal(t, repo.NumClosedProjects, repoAfter.NumClosedProjects)
	})
}

func TestGetProjectForRepoByIDOrTitle(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Test getting by ID
	project, err := GetProjectForRepoByIDOrTitle(db.DefaultContext, 1, "1")
	require.NoError(t, err)
	assert.Equal(t, int64(1), project.ID)

	// Test getting by title
	project, err = GetProjectForRepoByIDOrTitle(db.DefaultContext, 1, "First project")
	require.NoError(t, err)
	assert.Equal(t, "First project", project.Title)

	// Test non-existent project
	_, err = GetProjectForRepoByIDOrTitle(db.DefaultContext, 1, "nonexistent")
	require.Error(t, err)
	assert.True(t, IsErrProjectNotExist(err))
}

func TestProjectState(t *testing.T) {
	project := &Project{IsClosed: false}
	assert.Equal(t, "open", string(project.State()))

	project.IsClosed = true
	assert.Equal(t, "closed", string(project.State()))
}

func TestGetProjectsByIDs(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("MultipleIDs", func(t *testing.T) {
		projects, err := GetProjectsByIDs(db.DefaultContext, []int64{1, 2, 3})
		require.NoError(t, err)
		assert.Len(t, projects, 3)
		assert.Equal(t, "First project", projects[1].Title)
	})

	t.Run("PartialMatch", func(t *testing.T) {
		projects, err := GetProjectsByIDs(db.DefaultContext, []int64{1, 99999})
		require.NoError(t, err)
		assert.Len(t, projects, 1)
		assert.NotNil(t, projects[1])
	})

	t.Run("EmptyInput", func(t *testing.T) {
		projects, err := GetProjectsByIDs(db.DefaultContext, []int64{})
		require.NoError(t, err)
		assert.Empty(t, projects)
	})
}

func TestGetProjectForOrgByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create an org project for testing
	orgProject := &Project{
		Type:        TypeOrganization,
		CardType:    CardTypeTextOnly,
		Title:       "Org Test Project",
		OwnerID:     3, // org3 in fixtures
		CreatedUnix: timeutil.TimeStampNow(),
		CreatorID:   2,
	}
	require.NoError(t, NewProject(db.DefaultContext, orgProject))

	t.Run("Success", func(t *testing.T) {
		p, err := GetProjectForOrgByID(db.DefaultContext, 3, orgProject.ID)
		require.NoError(t, err)
		assert.Equal(t, orgProject.ID, p.ID)
	})

	t.Run("WrongOrg", func(t *testing.T) {
		_, err := GetProjectForOrgByID(db.DefaultContext, 999, orgProject.ID)
		require.Error(t, err)
		assert.True(t, IsErrProjectNotExist(err))
	})

	t.Run("NonExistent", func(t *testing.T) {
		_, err := GetProjectForOrgByID(db.DefaultContext, 3, 99999)
		require.Error(t, err)
		assert.True(t, IsErrProjectNotExist(err))
	})
}

func TestGetProjectForOrgByIDOrTitle(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Create an org project for testing
	orgProject := &Project{
		Type:        TypeOrganization,
		CardType:    CardTypeTextOnly,
		Title:       "Org Lookup Project",
		OwnerID:     3,
		CreatedUnix: timeutil.TimeStampNow(),
		CreatorID:   2,
	}
	require.NoError(t, NewProject(db.DefaultContext, orgProject))

	t.Run("ByID", func(t *testing.T) {
		p, err := GetProjectForOrgByIDOrTitle(db.DefaultContext, 3, fmt.Sprintf("%d", orgProject.ID))
		require.NoError(t, err)
		assert.Equal(t, orgProject.ID, p.ID)
	})

	t.Run("ByTitle", func(t *testing.T) {
		p, err := GetProjectForOrgByIDOrTitle(db.DefaultContext, 3, "Org Lookup Project")
		require.NoError(t, err)
		assert.Equal(t, orgProject.Title, p.Title)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, err := GetProjectForOrgByIDOrTitle(db.DefaultContext, 3, "nonexistent")
		require.Error(t, err)
		assert.True(t, IsErrProjectNotExist(err))
	})
}

func TestProjectsSortAlphabetical(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Test alphabetical sort
	projects, _, err := db.FindAndCount[Project](db.DefaultContext, SearchOptions{
		OrderBy: GetSearchOrderByBySortType("alphabetically"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, projects)
	// Verify titles are in ascending order
	for i := 1; i < len(projects); i++ {
		assert.LessOrEqual(t, projects[i-1].Title, projects[i].Title)
	}

	// Test reverse alphabetical sort
	projects, _, err = db.FindAndCount[Project](db.DefaultContext, SearchOptions{
		OrderBy: GetSearchOrderByBySortType("reversealphabetically"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, projects)
	// Verify titles are in descending order
	for i := 1; i < len(projects); i++ {
		assert.GreaterOrEqual(t, projects[i-1].Title, projects[i].Title)
	}
}

func TestIsValidSortType(t *testing.T) {
	assert.True(t, IsValidSortType("oldest"))
	assert.True(t, IsValidSortType("newest"))
	assert.True(t, IsValidSortType("alphabetically"))
	assert.True(t, IsValidSortType("reversealphabetically"))
	assert.True(t, IsValidSortType("recentupdate"))
	assert.True(t, IsValidSortType("leastupdate"))
	assert.True(t, IsValidSortType(""))
	assert.False(t, IsValidSortType("invalid"))
}

func TestDeleteProjectByID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Success", func(t *testing.T) {
		// Create a fresh project to delete
		p := &Project{
			Type:        TypeRepository,
			CardType:    CardTypeTextOnly,
			Title:       "Project To Delete",
			RepoID:      1,
			CreatedUnix: timeutil.TimeStampNow(),
			CreatorID:   2,
		}
		require.NoError(t, NewProject(db.DefaultContext, p))
		projectID := p.ID

		// Add a card to it
		columns, err := p.GetColumns(db.DefaultContext)
		require.NoError(t, err)
		if len(columns) > 0 {
			require.NoError(t, AddIssueToProject(db.DefaultContext, projectID, 4, columns[0].ID, -1))
		}

		// Delete it
		require.NoError(t, DeleteProjectByID(db.DefaultContext, projectID))

		// Should not exist
		_, err = GetProjectByID(db.DefaultContext, projectID)
		require.Error(t, err)
		assert.True(t, IsErrProjectNotExist(err))
	})

	t.Run("NonExistentIsSilent", func(t *testing.T) {
		err := DeleteProjectByID(db.DefaultContext, 99999)
		require.NoError(t, err)
	})
}

func TestChangeProjectStatusRoundTrip(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	p := &Project{
		Type:        TypeRepository,
		CardType:    CardTypeTextOnly,
		Title:       "Status Test Project",
		RepoID:      1,
		CreatedUnix: timeutil.TimeStampNow(),
		CreatorID:   2,
	}
	require.NoError(t, NewProject(db.DefaultContext, p))

	// Close it
	require.NoError(t, ChangeProjectStatus(db.DefaultContext, p, true))
	closed, err := GetProjectByID(db.DefaultContext, p.ID)
	require.NoError(t, err)
	assert.True(t, closed.IsClosed)
	assert.Positive(t, int64(closed.ClosedDateUnix))

	// Reopen — ClosedDateUnix should be cleared
	require.NoError(t, ChangeProjectStatus(db.DefaultContext, closed, false))
	reopened, err := GetProjectByID(db.DefaultContext, p.ID)
	require.NoError(t, err)
	assert.False(t, reopened.IsClosed)
	assert.Equal(t, int64(0), int64(reopened.ClosedDateUnix))
}

func TestCanBeAccessedByOwnerRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("RepoProject", func(t *testing.T) {
		p := unittest.AssertExistsAndLoadBean(t, &Project{ID: 1})
		require.Equal(t, TypeRepository, p.Type)

		// Matching repo
		assert.True(t, p.CanBeAccessedByOwnerRepo(0, &repo_model.Repository{ID: p.RepoID}))

		// Wrong repo
		assert.False(t, p.CanBeAccessedByOwnerRepo(0, &repo_model.Repository{ID: 99999}))

		// Nil repo
		assert.False(t, p.CanBeAccessedByOwnerRepo(0, nil))
	})

	t.Run("OrgProject", func(t *testing.T) {
		// Create an org project
		p := &Project{
			Type:        TypeOrganization,
			CardType:    CardTypeTextOnly,
			Title:       "Org Access Test",
			OwnerID:     3,
			CreatedUnix: timeutil.TimeStampNow(),
			CreatorID:   2,
		}
		require.NoError(t, NewProject(db.DefaultContext, p))

		// Matching owner
		assert.True(t, p.CanBeAccessedByOwnerRepo(3, nil))

		// Wrong owner
		assert.False(t, p.CanBeAccessedByOwnerRepo(99999, nil))
	})
}
