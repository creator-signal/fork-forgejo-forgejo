package edu

import (
	"context"
	"fmt"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
)

// CreateAssignmentOptions contains options for creating a new assignment.
type CreateAssignmentOptions struct {
	RepoID       int64
	Title        string
	Description  string
	DeadlineUnix int64
}

// EducationalService defines the business logic for the educational module.
type EducationalService interface {
	CreateAssignment(ctx context.Context, opts CreateAssignmentOptions) (*Assignment, error)
	GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error)
	GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error)
	GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error)
	JoinAssignment(ctx context.Context, doer *user_model.User, assignmentID int64) (*Submission, error)
}

// RepoForker abstracts the repository forking and retrieval logic.
type RepoForker interface {
	ForkRepositoryAndUpdates(ctx context.Context, doer, owner *user_model.User, opts ForkRepoOptions) (*repo_model.Repository, error)
	GetRepositoryByID(ctx context.Context, id int64) (*repo_model.Repository, error)
}

// ForkRepoOptions is a subset of options needed for forking.
type ForkRepoOptions struct {
	BaseRepo *repo_model.Repository
	Name     string
}

// private implementation
type service struct {
	repo   Repository
	forker RepoForker
}

// Repository defines the data access layer interface.
type Repository interface {
	CreateAssignment(ctx context.Context, assignment *Assignment) error
	GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error)
	GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error)
	CreateSubmission(ctx context.Context, submission *Submission) error
	GetSubmission(ctx context.Context, assignmentID, userID int64) (*Submission, error)
	GetSubmissionByRepoID(ctx context.Context, repoID int64) (*Submission, error)
	GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error)
	UpdateSubmission(ctx context.Context, submission *Submission) error
}

// NewService creates a new instance of EducationalService.
func NewService(repo Repository, forker RepoForker) EducationalService {
	RegisterNotifier(repo)
	return &service{repo: repo, forker: forker}
}

func (s *service) CreateAssignment(ctx context.Context, opts CreateAssignmentOptions) (*Assignment, error) {
	if opts.Title == "" {
		return nil, fmt.Errorf("title is required")
	}

	assignment := &Assignment{
		RepoID:       opts.RepoID,
		Title:        opts.Title,
		Description:  opts.Description,
		DeadlineUnix: opts.DeadlineUnix,
	}

	if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
		return nil, fmt.Errorf("failed to create assignment: %w", err)
	}

	return assignment, nil
}

func (s *service) GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error) {
	return s.repo.GetAssignmentByID(ctx, id)
}

func (s *service) GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error) {
	return s.repo.GetSubmissions(ctx, assignmentID)
}
