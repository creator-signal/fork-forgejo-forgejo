package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateAssignment_StoresTaskNameAndGlob(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	a := &Assignment{
		CourseID:         1,
		TaskName:         "multiplication",
		AllowedFilesGlob: "tasks/multiplication/multiplication.cpp",
		Title:            "Умножение",
		DeadlineUnix:     0,
	}
	assert.NoError(t, repo.CreateAssignment(ctx, a))

	got, err := repo.GetAssignmentByID(ctx, a.ID)
	assert.NoError(t, err)
	assert.Equal(t, "multiplication", got.TaskName)
	assert.Equal(t, "tasks/multiplication/multiplication.cpp", got.AllowedFilesGlob)
}

func TestRepository_GetAssignmentByCourseAndTask(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	a := &Assignment{CourseID: 1, TaskName: "stack", AllowedFilesGlob: "tasks/stack/stack.cpp", Title: "Stack"}
	assert.NoError(t, repo.CreateAssignment(ctx, a))

	got, err := repo.GetAssignmentByCourseAndTask(ctx, 1, "stack")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, a.ID, got.ID)
}

func TestRepository_CreateAssignment_DuplicateTaskNameInCourse_Fails(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	a1 := &Assignment{CourseID: 1, TaskName: "stack", AllowedFilesGlob: "tasks/stack/*.cpp", Title: "S1"}
	assert.NoError(t, repo.CreateAssignment(ctx, a1))

	a2 := &Assignment{CourseID: 1, TaskName: "stack", AllowedFilesGlob: "tasks/stack/*.cpp", Title: "S2"}
	err := repo.CreateAssignment(ctx, a2)
	assert.Error(t, err)
}
