package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateCourseSyncTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	tk := &CourseSyncTask{
		CourseID:   1,
		CreatorID:  7,
		TotalRepos: 10,
		Status:     StatusPending,
	}
	assert.NoError(t, repo.CreateCourseSyncTask(ctx, tk))
	assert.Greater(t, tk.ID, int64(0))
}
