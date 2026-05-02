package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateInitForksTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	tk := &InitForksTask{
		CourseID:   1,
		CreatorID:  7,
		TotalUsers: 10,
		Status:     StatusPending,
	}
	assert.NoError(t, repo.CreateInitForksTask(ctx, tk))
	assert.Greater(t, tk.ID, int64(0))
}
