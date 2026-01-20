package edu

import (
	"context"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/repository"
)

// ForgejoAdapter implements RepoForker using Forgejo's internal services.
type ForgejoAdapter struct{}

// NewForgejoAdapter creates a new ForgejoAdapter.
func NewForgejoAdapter() *ForgejoAdapter {
	return &ForgejoAdapter{}
}

// ForkRepositoryAndUpdates calls the Forgejo service to fork a repository.
func (a *ForgejoAdapter) ForkRepositoryAndUpdates(ctx context.Context, doer, owner *user_model.User, opts ForkRepoOptions) (*repo_model.Repository, error) {
	serviceOpts := repository.ForkRepoOptions{
		BaseRepo:    opts.BaseRepo,
		Name:        opts.Name,
		Description: opts.BaseRepo.Description,
	}
	return repository.ForkRepositoryAndUpdates(ctx, doer, owner, serviceOpts)
}

// GetRepositoryByID retrieves a repository by ID.
func (a *ForgejoAdapter) GetRepositoryByID(ctx context.Context, id int64) (*repo_model.Repository, error) {
	return repo_model.GetRepositoryByID(ctx, id)
}
