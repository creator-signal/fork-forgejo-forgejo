package edu

import (
	"context"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"github.com/stretchr/testify/mock"
)

// MockRepository is a mock implementation of Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) CreateAssignment(ctx context.Context, assignment *Assignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockRepository) GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Assignment), args.Error(1)
}

func (m *MockRepository) GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Assignment), args.Error(1)
}

func (m *MockRepository) CreateSubmission(ctx context.Context, submission *Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockRepository) GetSubmission(ctx context.Context, assignmentID, userID int64) (*Submission, error) {
	args := m.Called(ctx, assignmentID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Submission), args.Error(1)
}

func (m *MockRepository) GetSubmissionByRepoID(ctx context.Context, repoID int64) (*Submission, error) {
	args := m.Called(ctx, repoID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Submission), args.Error(1)
}

func (m *MockRepository) UpdateSubmission(ctx context.Context, submission *Submission) error {
	args := m.Called(ctx, submission)
	return args.Error(0)
}

func (m *MockRepository) GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error) {
	args := m.Called(ctx, assignmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Submission), args.Error(1)
}

// MockRepoForker mocks the RepoForker interface
type MockRepoForker struct {
	mock.Mock
}

func (m *MockRepoForker) ForkRepositoryAndUpdates(ctx context.Context, doer, owner *user_model.User, opts ForkRepoOptions) (*repo_model.Repository, error) {
	args := m.Called(ctx, doer, owner, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo_model.Repository), args.Error(1)
}

func (m *MockRepoForker) GetRepositoryByID(ctx context.Context, id int64) (*repo_model.Repository, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repo_model.Repository), args.Error(1)
}
