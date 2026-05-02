package edu

import (
	"context"
	"fmt"

	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	pull_service "forgejo.org/services/pull"
)

// CreatePullRequestOptions describes a PR creation request.
type CreatePullRequestOptions struct {
	BaseRepoID int64
	BaseBranch string
	HeadRepoID int64
	HeadBranch string
	Title      string
	Body       string
	Doer       *user_model.User
}

// CreatePullRequest creates a PR HeadRepo:HeadBranch -> BaseRepo:BaseBranch.
func (a *ForgejoAdapter) CreatePullRequest(ctx context.Context, opts CreatePullRequestOptions) (*issues_model.PullRequest, error) {
	baseRepo, err := repo_model.GetRepositoryByID(ctx, opts.BaseRepoID)
	if err != nil {
		return nil, fmt.Errorf("load base repo: %w", err)
	}
	headRepo, err := repo_model.GetRepositoryByID(ctx, opts.HeadRepoID)
	if err != nil {
		return nil, fmt.Errorf("load head repo: %w", err)
	}

	pr := &issues_model.PullRequest{
		HeadRepoID: headRepo.ID,
		BaseRepoID: baseRepo.ID,
		HeadBranch: opts.HeadBranch,
		BaseBranch: opts.BaseBranch,
		HeadRepo:   headRepo,
		BaseRepo:   baseRepo,
		Type:       issues_model.PullRequestGitea,
	}
	prIssue := &issues_model.Issue{
		RepoID:   baseRepo.ID,
		Title:    opts.Title,
		PosterID: opts.Doer.ID,
		Poster:   opts.Doer,
		IsPull:   true,
		Content:  opts.Body,
	}

	// NewPullRequest(ctx, repo, issue, labelIDs, uuids, pr, assigneeIDs)
	if err := pull_service.NewPullRequest(ctx, baseRepo, prIssue, nil, nil, pr, nil); err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	return pr, nil
}
