package edu

import (
	"context"
	"testing"
	"time"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetAssignmentsForUser_Service(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	expected := []*Assignment{{ID: 1, CourseID: 10, Title: "HW1"}}
	mockRepo.On("GetAssignmentsForUser", ctx, int64(5)).Return(expected, nil)

	result, err := svc.GetAssignmentsForUser(ctx, 5)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestUpdateAssignment_Service(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		a := &Assignment{ID: 1, Title: "Updated"}
		mockRepo.On("UpdateAssignment", ctx, mock.MatchedBy(func(a *Assignment) bool {
			return a.ID == int64(1) && a.Title == "Updated" && a.UpdatedUnix > 0
		})).Return(nil).Once()

		err := svc.UpdateAssignment(ctx, a)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Empty title", func(t *testing.T) {
		a := &Assignment{ID: 1, Title: ""}
		err := svc.UpdateAssignment(ctx, a)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
}

func TestDeleteAssignment_Service(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	mockRepo.On("DeleteAssignment", ctx, int64(1)).Return(nil)

	err := svc.DeleteAssignment(ctx, 1)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestJoinAssignment_DeadlinePassed(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	doer := &user_model.User{ID: 10, Name: "student"}
	pastDeadline := time.Now().Add(-24 * time.Hour).Unix()
	assignment := &Assignment{ID: 1, RepoID: 100, DeadlineUnix: pastDeadline}

	mockRepo.On("GetAssignmentByID", ctx, assignment.ID).Return(assignment, nil)

	_, err := svc.JoinAssignment(ctx, doer, assignment.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline has passed")
}

func TestJoinAssignment_NotEnrolled(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	doer := &user_model.User{ID: 10, Name: "student"}
	assignment := &Assignment{ID: 1, RepoID: 100, CourseID: 5}

	mockRepo.On("GetAssignmentByID", ctx, assignment.ID).Return(assignment, nil)
	mockRepo.On("GetEnrollment", ctx, int64(5), int64(10)).Return(nil, nil)

	_, err := svc.JoinAssignment(ctx, doer, assignment.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enrolled")
}

func TestJoinAssignment_WithEnrollment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	doer := &user_model.User{ID: 10, Name: "student"}
	assignment := &Assignment{ID: 1, RepoID: 100, CourseID: 5}
	enrollment := &CourseEnrollment{ID: 1, CourseID: 5, UserID: 10, Role: RoleStudent}
	baseRepo := &repo_model.Repository{ID: 100, Name: "hw1-template"}
	forkedRepo := &repo_model.Repository{ID: 200, Name: "student-hw1-template"}

	mockRepo.On("GetAssignmentByID", ctx, assignment.ID).Return(assignment, nil)
	mockRepo.On("GetEnrollment", ctx, int64(5), int64(10)).Return(enrollment, nil)
	mockRepo.On("GetSubmission", ctx, assignment.ID, doer.ID).Return(nil, nil)
	mockForker.On("GetRepositoryByID", ctx, assignment.RepoID).Return(baseRepo, nil)
	mockForker.On("ForkRepositoryAndUpdates", ctx, doer, doer, mock.MatchedBy(func(opts ForkRepoOptions) bool {
		return opts.BaseRepo.ID == baseRepo.ID
	})).Return(forkedRepo, nil)
	mockRepo.On("CreateSubmission", ctx, mock.MatchedBy(func(s *Submission) bool {
		return s.AssignmentID == assignment.ID && s.UserID == doer.ID
	})).Return(nil)

	submission, err := svc.JoinAssignment(ctx, doer, assignment.ID)
	assert.NoError(t, err)
	assert.NotNil(t, submission)
	assert.Equal(t, forkedRepo.ID, submission.StudentRepoID)
	mockRepo.AssertExpectations(t)
}
