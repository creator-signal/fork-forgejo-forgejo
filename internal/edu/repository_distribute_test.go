package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateDistributeTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	tk := &DistributeTask{
		AssignmentID:     10,
		CreatorID:        7,
		TotalEnrollments: 20,
		Status:           StatusPending,
	}
	assert.NoError(t, repo.CreateDistributeTask(ctx, tk))
	assert.Greater(t, tk.ID, int64(0))

	got, err := repo.GetDistributeTask(ctx, tk.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), got.AssignmentID)
}

func TestRepository_GetDistributeTaskByAssignment(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	tk := &DistributeTask{AssignmentID: 11, CreatorID: 7, Status: StatusPending}
	assert.NoError(t, repo.CreateDistributeTask(ctx, tk))

	got, err := repo.GetDistributeTaskByAssignment(ctx, 11)
	assert.NoError(t, err)
	assert.NotNil(t, got)
}
