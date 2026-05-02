package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateCourse_StoresTasksMasterRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	course := &Course{
		Name:              "Cxx",
		CreatorID:         1,
		OrgID:             42,
		TasksMasterRepoID: 101,
	}
	assert.NoError(t, repo.CreateCourse(ctx, course))

	got, err := repo.GetCourseByID(ctx, course.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(101), got.TasksMasterRepoID)
}
