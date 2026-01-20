package edu

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAssignments_Service(t *testing.T) {
	mockRepo := new(MockRepository)
	mockForker := new(MockRepoForker)
	svc := NewService(mockRepo, mockForker)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		expected := []*Assignment{{ID: 1, Title: "Task 1"}}
		mockRepo.On("GetAssignments", ctx, int64(99)).Return(expected, nil)

		result, err := svc.GetAssignments(ctx, 99)
		assert.NoError(t, err)
		assert.Equal(t, expected, result)
		mockRepo.AssertExpectations(t)
	})
}
