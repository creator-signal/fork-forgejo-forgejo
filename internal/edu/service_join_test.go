package edu

import (
	"context"
	"testing"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestJoinAssignment(t *testing.T) {
	ctx := context.Background()

	doer := &user_model.User{ID: 10, Name: "student"}
	assignment := &Assignment{ID: 1, RepoID: 100, Title: "HW1"}
	baseRepo := &repo_model.Repository{ID: 100, Name: "hw1-template"}
	forkedRepo := &repo_model.Repository{ID: 200, Name: "student-hw1-template"} // Fork name logic: doer-baseRepo

	t.Run("Success", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockForker := new(MockRepoForker)
		svc := NewService(mockRepo, mockForker)

		mockRepo.On("GetAssignmentByID", ctx, assignment.ID).Return(assignment, nil)

		mockRepo.On("GetSubmission", ctx, assignment.ID, doer.ID).Return(nil, nil)

		mockForker.On("GetRepositoryByID", ctx, assignment.RepoID).Return(baseRepo, nil)

		mockForker.On("ForkRepositoryAndUpdates", ctx, doer, doer, mock.MatchedBy(func(opts ForkRepoOptions) bool {
			return opts.BaseRepo.ID == baseRepo.ID && opts.Name == "student-hw1-template"
		})).Return(forkedRepo, nil)

		mockRepo.On("CreateSubmission", ctx, mock.MatchedBy(func(s *Submission) bool {
			return s.AssignmentID == assignment.ID && s.UserID == doer.ID && s.StudentRepoID == forkedRepo.ID && s.Status == "started"
		})).Return(nil)

		submission, err := svc.JoinAssignment(ctx, doer, assignment.ID)
		assert.NoError(t, err)
		assert.NotNil(t, submission)
		assert.Equal(t, forkedRepo.ID, submission.StudentRepoID)
	})

	t.Run("AlreadyJoined", func(t *testing.T) {
		mockRepo := new(MockRepository)
		mockForker := new(MockRepoForker)
		svc := NewService(mockRepo, mockForker)

		existing := &Submission{ID: 55, Status: "started"}

		mockRepo.On("GetAssignmentByID", ctx, assignment.ID).Return(assignment, nil)
		mockRepo.On("GetSubmission", ctx, assignment.ID, doer.ID).Return(existing, nil)

		submission, err := svc.JoinAssignment(ctx, doer, assignment.ID)
		assert.NoError(t, err)
		assert.Equal(t, existing, submission)
	})
}
