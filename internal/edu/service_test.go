package edu

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateAssignment(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		opts := CreateAssignmentOptions{
			RepoID:       1,
			Title:        "Test Assignment",
			Description:  "Description",
			DeadlineUnix: time.Now().Add(24 * time.Hour).Unix(),
		}

		mockRepo.On("CreateAssignment", ctx, mock.MatchedBy(func(a *Assignment) bool {
			return a.RepoID == opts.RepoID && a.Title == opts.Title
		})).Return(nil)

		assignment, err := service.CreateAssignment(ctx, opts)
		assert.NoError(t, err)
		assert.NotNil(t, assignment)
		assert.Equal(t, opts.Title, assignment.Title)
		mockRepo.AssertExpectations(t)
	})

	t.Run("Validation Error", func(t *testing.T) {
		opts := CreateAssignmentOptions{RepoID: 1}
		_, err := service.CreateAssignment(ctx, opts)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "title is required")
	})
}

func TestGetAssignments(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	expected := []*Assignment{{ID: 1, Title: "A1"}}
	mockRepo.On("GetAssignments", ctx, int64(0)).Return(expected, nil)

	result, err := service.GetAssignments(ctx, 0)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}

func TestGetSubmissions(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	service := NewService(mockRepo, mockForker)
	ctx := context.Background()

	assignmentID := int64(100)
	expected := []*Submission{{ID: 1, Status: StatusSubmissionPending}}

	mockRepo.On("GetSubmissions", ctx, assignmentID).Return(expected, nil)

	result, err := service.GetSubmissions(ctx, assignmentID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	mockRepo.AssertExpectations(t)
}
