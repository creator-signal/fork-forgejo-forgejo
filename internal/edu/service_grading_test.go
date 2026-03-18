package edu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGradeSubmission_Valid(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	mockRepo.On("GradeSubmission", mock.Anything, int64(1), 85, "Good work", int64(50)).Return(nil)

	err := svc.GradeSubmission(context.Background(), 1, 85, "Good work", 50)
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestGradeSubmission_Zero(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	mockRepo.On("GradeSubmission", mock.Anything, int64(1), 0, "", int64(50)).Return(nil)

	err := svc.GradeSubmission(context.Background(), 1, 0, "", 50)
	assert.NoError(t, err)
}

func TestGradeSubmission_MaxScore(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	mockRepo.On("GradeSubmission", mock.Anything, int64(1), 100, "Perfect", int64(50)).Return(nil)

	err := svc.GradeSubmission(context.Background(), 1, 100, "Perfect", 50)
	assert.NoError(t, err)
}

func TestGradeSubmission_TooHigh(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	err := svc.GradeSubmission(context.Background(), 1, 101, "", 50)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grade must be between 0 and 100")
}

func TestGradeSubmission_Negative(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	err := svc.GradeSubmission(context.Background(), 1, -1, "", 50)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "grade must be between 0 and 100")
}

func TestGetTestResults(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	results := []*TestResult{
		{ID: 1, SubmissionID: 10, CommitSHA: "abc1234", Score: 100},
		{ID: 2, SubmissionID: 10, CommitSHA: "def5678", Score: 0},
	}
	mockRepo.On("GetTestResultsBySubmission", mock.Anything, int64(10)).Return(results, nil)

	got, err := svc.GetTestResults(context.Background(), 10)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestGetLatestTestResult(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)

	tr := &TestResult{ID: 5, SubmissionID: 10, CommitSHA: "abc1234", Score: 100}
	mockRepo.On("GetLatestTestResult", mock.Anything, int64(10)).Return(tr, nil)

	got, err := svc.GetLatestTestResult(context.Background(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, int64(5), got.ID)
	assert.Equal(t, 100, got.Score)
}
