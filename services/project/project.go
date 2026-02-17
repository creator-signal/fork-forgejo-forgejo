// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"

	org_model "forgejo.org/models/organization"
	project_model "forgejo.org/models/project"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
)

// CreateProjectOptions represents options for creating a project
type CreateProjectOptions struct {
	Title        string
	Description  string
	TemplateType project_model.TemplateType
	CardType     project_model.CardType
	CanWrite     bool // Whether the creator has write permissions
}

// UpdateProjectOptions represents options for updating a project
type UpdateProjectOptions struct {
	Title       *string
	Description *string
	CardType    *project_model.CardType
	IsClosed    *bool
}

// Owner represents something that can own projects
type Owner interface {
	GetID() int64
	GetProjectType() project_model.Type
	GetOwnerID() int64 // for orgs/users
	GetRepoID() int64  // for repos
}

// RepoOwner wraps a repository as a project owner
type RepoOwner struct {
	Repo *repo_model.Repository
}

func (r RepoOwner) GetID() int64                       { return r.Repo.ID }
func (r RepoOwner) GetProjectType() project_model.Type { return project_model.TypeRepository }
func (r RepoOwner) GetOwnerID() int64                  { return 0 }
func (r RepoOwner) GetRepoID() int64                   { return r.Repo.ID }

// OrgOwner wraps an organization as a project owner
type OrgOwner struct {
	Org *org_model.Organization
}

func (o OrgOwner) GetID() int64                       { return o.Org.ID }
func (o OrgOwner) GetProjectType() project_model.Type { return project_model.TypeOrganization }
func (o OrgOwner) GetOwnerID() int64                  { return o.Org.ID }
func (o OrgOwner) GetRepoID() int64                   { return 0 }

// CreateProjectGeneric creates a project for any owner type
func CreateProjectGeneric(ctx context.Context, owner Owner, creator *user_model.User, opts CreateProjectOptions) (*project_model.Project, error) {
	project := &project_model.Project{
		Title:       opts.Title,
		Description: opts.Description,
		CreatorID:   creator.ID,
		Type:        owner.GetProjectType(),
	}

	// Set owner fields based on type
	if owner.GetProjectType() == project_model.TypeRepository {
		project.RepoID = owner.GetRepoID()
	} else {
		project.OwnerID = owner.GetOwnerID()
	}

	// Apply permission-based field restrictions
	if opts.CanWrite {
		project.TemplateType = opts.TemplateType
		project.CardType = opts.CardType
	} else {
		project.TemplateType = project_model.TemplateTypeBasicKanban
		project.CardType = project_model.CardTypeTextOnly
	}

	return project, project_model.NewProject(ctx, project)
}

// CreateOrgProject creates an organization project
func CreateOrgProject(ctx context.Context, org *org_model.Organization, creator *user_model.User, opts CreateProjectOptions) (*project_model.Project, error) {
	return CreateProjectGeneric(ctx, OrgOwner{org}, creator, opts)
}

// ValidateProjectOwner checks if a project belongs to the given owner
func ValidateProjectOwner(project *project_model.Project, owner Owner) bool {
	if project.Type != owner.GetProjectType() {
		return false
	}

	switch project.Type {
	case project_model.TypeRepository:
		return project.RepoID == owner.GetRepoID()
	case project_model.TypeOrganization:
		return project.OwnerID == owner.GetOwnerID()
	default:
		return false
	}
}

// CreateProject creates a new repository project (backwards compatible)
func CreateProject(ctx context.Context, repo *repo_model.Repository, creator *user_model.User, opts CreateProjectOptions) (*project_model.Project, error) {
	return CreateProjectGeneric(ctx, RepoOwner{repo}, creator, opts)
}

// UpdateProject updates an existing project with proper validation
func UpdateProject(ctx context.Context, project *project_model.Project, opts UpdateProjectOptions) error {
	if opts.Title != nil {
		project.Title = *opts.Title
	}
	if opts.Description != nil {
		project.Description = *opts.Description
	}
	if opts.CardType != nil {
		project.CardType = *opts.CardType
	}
	if opts.IsClosed != nil && *opts.IsClosed != project.IsClosed {
		if err := project_model.ChangeProjectStatus(ctx, project, *opts.IsClosed); err != nil {
			return err
		}
		// Re-read to get exact persisted values
		updated, err := project_model.GetProjectByID(ctx, project.ID)
		if err != nil {
			return err
		}
		project.IsClosed = updated.IsClosed
		project.ClosedDateUnix = updated.ClosedDateUnix
	}

	return project_model.UpdateProject(ctx, project)
}

// DeleteProject deletes a project with proper cleanup
func DeleteProject(ctx context.Context, project *project_model.Project) error {
	return project_model.DeleteProjectByID(ctx, project.ID)
}

// ChangeProjectStatus changes the open/closed status of a project
func ChangeProjectStatus(ctx context.Context, project *project_model.Project, isClosed bool) error {
	return project_model.ChangeProjectStatus(ctx, project, isClosed)
}
