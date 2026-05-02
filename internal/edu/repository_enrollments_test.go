package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_EnrollUser_StoresGroupAndForkRepoID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	e := &CourseEnrollment{
		CourseID:          1,
		UserID:            7,
		Role:              RoleStudent,
		GroupName:         "se241",
		StudentForkRepoID: 555,
	}
	assert.NoError(t, repo.EnrollUser(ctx, e))

	got, err := repo.GetEnrollmentByCourseUser(ctx, 1, 7)
	assert.NoError(t, err)
	assert.Equal(t, "se241", got.GroupName)
	assert.Equal(t, int64(555), got.StudentForkRepoID)
}
