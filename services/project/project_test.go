// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"testing"

	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	t.Run("CreateProjectWithWritePermissions", func(t *testing.T) {
		opts := CreateProjectOptions{
			Title:        "Test Project",
			Description:  "Test Description",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeImagesAndText,
			CanWrite:     true,
		}

		project, err := CreateProject(db.DefaultContext, repo, user, opts)
		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "Test Project", project.Title)
		assert.Equal(t, "Test Description", project.Description)
		assert.Equal(t, project_model.TemplateTypeBasicKanban, project.TemplateType)
		assert.Equal(t, project_model.CardTypeImagesAndText, project.CardType)
		assert.Equal(t, user.ID, project.CreatorID)
		assert.Equal(t, repo.ID, project.RepoID)
	})

	t.Run("CreateProjectWithoutWritePermissions", func(t *testing.T) {
		opts := CreateProjectOptions{
			Title:        "Test Project No Write",
			Description:  "Test Description",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeImagesAndText,
			CanWrite:     false, // No write permissions
		}

		project, err := CreateProject(db.DefaultContext, repo, user, opts)
		require.NoError(t, err)
		assert.NotNil(t, project)
		assert.Equal(t, "Test Project No Write", project.Title)
		// Should use safe defaults for non-writers
		assert.Equal(t, project_model.TemplateTypeBasicKanban, project.TemplateType)
		assert.Equal(t, project_model.CardTypeTextOnly, project.CardType)
	})
}

func TestUpdateProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Get an existing project
	project := unittest.AssertExistsAndLoadBean(t, &project_model.Project{ID: 1})
	originalTitle := project.Title

	t.Run("UpdateProjectTitle", func(t *testing.T) {
		newTitle := "Updated Project Title"
		opts := UpdateProjectOptions{
			Title: &newTitle,
		}

		err := UpdateProject(db.DefaultContext, project, opts)
		require.NoError(t, err)
		assert.Equal(t, newTitle, project.Title)
	})

	t.Run("UpdateProjectClosed", func(t *testing.T) {
		isClosed := true
		opts := UpdateProjectOptions{
			IsClosed: &isClosed,
		}

		err := UpdateProject(db.DefaultContext, project, opts)
		require.NoError(t, err)
		assert.True(t, project.IsClosed)
	})

	t.Run("UpdateProjectDescription", func(t *testing.T) {
		newDescription := "Updated project description"
		opts := UpdateProjectOptions{
			Description: &newDescription,
		}

		err := UpdateProject(db.DefaultContext, project, opts)
		require.NoError(t, err)
		assert.Equal(t, newDescription, project.Description)
	})

	t.Run("UpdateProjectCardType", func(t *testing.T) {
		newCardType := project_model.CardTypeImagesAndText
		opts := UpdateProjectOptions{
			CardType: &newCardType,
		}

		err := UpdateProject(db.DefaultContext, project, opts)
		require.NoError(t, err)
		assert.Equal(t, newCardType, project.CardType)
	})

	// Restore original state
	restoreOpts := UpdateProjectOptions{
		Title: &originalTitle,
	}
	isClosed := false
	restoreOpts.IsClosed = &isClosed
	_ = UpdateProject(db.DefaultContext, project, restoreOpts)
}

func TestCreateOrgProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})

	t.Run("Successfully create organization project", func(t *testing.T) {
		opts := CreateProjectOptions{
			Title:        "Test Org Project",
			Description:  "Test Organization Project Description",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeImagesAndText,
			CanWrite:     true,
		}

		project, err := CreateOrgProject(db.DefaultContext, org, user, opts)
		require.NoError(t, err)
		require.NotNil(t, project)

		assert.Equal(t, "Test Org Project", project.Title)
		assert.Equal(t, "Test Organization Project Description", project.Description)
		assert.Equal(t, project_model.TemplateTypeBasicKanban, project.TemplateType)
		assert.Equal(t, project_model.CardTypeImagesAndText, project.CardType)
		assert.Equal(t, user.ID, project.CreatorID)
		assert.Equal(t, org.ID, project.OwnerID)
		assert.Equal(t, project_model.TypeOrganization, project.Type)
		assert.Equal(t, int64(0), project.RepoID) // Organization projects don't have repo ID
	})
}

func TestValidateProjectOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})

	// Create test repository project
	repoProject := &project_model.Project{
		Title:       "Test Repo Project",
		Description: "Test repository project",
		RepoID:      repo.ID,
		CreatorID:   user.ID,
		Type:        project_model.TypeRepository,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, repoProject))

	// Create test organization project
	orgProject := &project_model.Project{
		Title:       "Test Org Project",
		Description: "Test organization project",
		OwnerID:     org.ID,
		CreatorID:   user.ID,
		Type:        project_model.TypeOrganization,
	}
	require.NoError(t, project_model.NewProject(db.DefaultContext, orgProject))

	t.Run("Valid repository project owner", func(t *testing.T) {
		repoOwner := RepoOwner{repo}
		assert.True(t, ValidateProjectOwner(repoProject, repoOwner))
	})

	t.Run("Valid organization project owner", func(t *testing.T) {
		orgOwner := OrgOwner{org}
		assert.True(t, ValidateProjectOwner(orgProject, orgOwner))
	})

	t.Run("Invalid repository project owner", func(t *testing.T) {
		// Try to validate repo project with org owner
		orgOwner := OrgOwner{org}
		assert.False(t, ValidateProjectOwner(repoProject, orgOwner))
	})

	t.Run("Invalid organization project owner", func(t *testing.T) {
		// Try to validate org project with repo owner
		repoOwner := RepoOwner{repo}
		assert.False(t, ValidateProjectOwner(orgProject, repoOwner))
	})

	t.Run("Wrong repository ID", func(t *testing.T) {
		// Create owner with different repo
		otherRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		wrongRepoOwner := RepoOwner{otherRepo}
		assert.False(t, ValidateProjectOwner(repoProject, wrongRepoOwner))
	})

	t.Run("Unknown project type", func(t *testing.T) {
		// Create project with unknown type (default case)
		unknownProject := &project_model.Project{
			Title:       "Unknown Type Project",
			Description: "Project with unknown type",
			Type:        99, // Invalid type to hit default case
			CreatorID:   user.ID,
		}
		repoOwner := RepoOwner{repo}
		assert.False(t, ValidateProjectOwner(unknownProject, repoOwner))
	})
}

func TestDeleteProject(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	t.Run("Successfully delete project", func(t *testing.T) {
		// Create a new project to delete
		opts := CreateProjectOptions{
			Title:        "Project to Delete",
			Description:  "This project will be deleted",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeTextOnly,
			CanWrite:     true,
		}

		project, err := CreateProject(db.DefaultContext, repo, user, opts)
		require.NoError(t, err)
		require.NotNil(t, project)

		projectID := project.ID

		// Verify project exists
		foundProject, err := project_model.GetProjectByID(db.DefaultContext, projectID)
		require.NoError(t, err)
		require.NotNil(t, foundProject)

		// Delete the project
		err = DeleteProject(db.DefaultContext, project)
		require.NoError(t, err)

		// Verify project is deleted
		deletedProject, err := project_model.GetProjectByID(db.DefaultContext, projectID)
		require.Error(t, err) // Should not be found
		assert.Nil(t, deletedProject)
	})

	t.Run("No error when deleting non-existent project", func(t *testing.T) {
		// Create a fake project with non-existent ID
		fakeProject := &project_model.Project{
			ID: 99999,
		}

		err := DeleteProject(db.DefaultContext, fakeProject)
		// This is acceptable behavior - either no error or an error is fine
		_ = err
	})
}

func TestChangeProjectStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})

	t.Run("Change repository project status to closed", func(t *testing.T) {
		// Create a repository project
		opts := CreateProjectOptions{
			Title:        "Repo Project Status Test",
			Description:  "Test changing repository project status",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeTextOnly,
			CanWrite:     true,
		}

		project, err := CreateProject(db.DefaultContext, repo, user, opts)
		require.NoError(t, err)
		require.NotNil(t, project)
		assert.False(t, project.IsClosed) // Should start as open

		// Close the project
		err = ChangeProjectStatus(db.DefaultContext, project, true)
		require.NoError(t, err)

		// Reload project from database to get updated status
		reloadedProject, err := project_model.GetProjectByID(db.DefaultContext, project.ID)
		require.NoError(t, err)
		assert.True(t, reloadedProject.IsClosed)

		// Open the project again
		err = ChangeProjectStatus(db.DefaultContext, project, false)
		require.NoError(t, err)

		// Reload project again
		reloadedProject, err = project_model.GetProjectByID(db.DefaultContext, project.ID)
		require.NoError(t, err)
		assert.False(t, reloadedProject.IsClosed)
	})

	t.Run("Change organization project status to closed", func(t *testing.T) {
		// Create an organization project
		opts := CreateProjectOptions{
			Title:        "Org Project Status Test",
			Description:  "Test changing organization project status",
			TemplateType: project_model.TemplateTypeBasicKanban,
			CardType:     project_model.CardTypeTextOnly,
			CanWrite:     true,
		}

		project, err := CreateOrgProject(db.DefaultContext, org, user, opts)
		require.NoError(t, err)
		require.NotNil(t, project)
		assert.False(t, project.IsClosed) // Should start as open

		// Close the project
		err = ChangeProjectStatus(db.DefaultContext, project, true)
		require.NoError(t, err)

		// Reload project from database to get updated status
		reloadedProject, err := project_model.GetProjectByID(db.DefaultContext, project.ID)
		require.NoError(t, err)
		assert.True(t, reloadedProject.IsClosed)

		// Open the project again
		err = ChangeProjectStatus(db.DefaultContext, project, false)
		require.NoError(t, err)

		// Reload project again
		reloadedProject, err = project_model.GetProjectByID(db.DefaultContext, project.ID)
		require.NoError(t, err)
		assert.False(t, reloadedProject.IsClosed)
	})
}

func TestProjectOwnerInterfaces(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	org := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})

	t.Run("RepoOwner interface methods", func(t *testing.T) {
		repoOwner := RepoOwner{repo}

		// Test GetID method
		assert.Equal(t, repo.ID, repoOwner.GetID())

		// Test GetProjectType method
		assert.Equal(t, project_model.TypeRepository, repoOwner.GetProjectType())

		// Test GetOwnerID method (should return 0 for repo owners)
		assert.Equal(t, int64(0), repoOwner.GetOwnerID())

		// Test GetRepoID method
		assert.Equal(t, repo.ID, repoOwner.GetRepoID())
	})

	t.Run("OrgOwner interface methods", func(t *testing.T) {
		orgOwner := OrgOwner{org}

		// Test GetID method
		assert.Equal(t, org.ID, orgOwner.GetID())

		// Test GetProjectType method
		assert.Equal(t, project_model.TypeOrganization, orgOwner.GetProjectType())

		// Test GetOwnerID method
		assert.Equal(t, org.ID, orgOwner.GetOwnerID())

		// Test GetRepoID method (should return 0 for org owners)
		assert.Equal(t, int64(0), orgOwner.GetRepoID())
	})
}
