package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateCourseSyncPR(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	pr := &CourseSyncPR{
		SyncTaskID:    100,
		EnrollmentID:  20,
		PullRequestID: 555,
		Status:        SyncPRStatusPending,
	}
	assert.NoError(t, repo.CreateCourseSyncPR(ctx, pr))
	assert.Greater(t, pr.ID, int64(0))
}

func TestRepository_ListCourseSyncPRsByTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	for i := 0; i < 3; i++ {
		assert.NoError(t, repo.CreateCourseSyncPR(ctx, &CourseSyncPR{
			SyncTaskID: 100, EnrollmentID: int64(i + 1), Status: SyncPRStatusPending,
		}))
	}
	got, err := repo.ListCourseSyncPRsByTask(ctx, 100)
	assert.NoError(t, err)
	assert.Len(t, got, 3)
}
