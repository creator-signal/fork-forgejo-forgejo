package edu

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
)

func TestRepository_CreateSubmission_StoresNewFields(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	s := &Submission{
		AssignmentID:  10,
		EnrollmentID:  20,
		UserID:        7,
		BranchName:    "submits/multiplication",
		PullRequestID: 0,
		Status:        StatusSubmissionPending,
	}
	assert.NoError(t, repo.CreateSubmission(ctx, s))

	got, err := repo.GetSubmissionByID(ctx, s.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(20), got.EnrollmentID)
	assert.Equal(t, "submits/multiplication", got.BranchName)
	assert.Equal(t, int64(0), got.PullRequestID)
}

func TestRepository_GetSubmissionByEnrollmentAssignment(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := NewRepository()
	ctx := db.DefaultContext

	s := &Submission{AssignmentID: 10, EnrollmentID: 20, UserID: 7, BranchName: "submits/x", Status: StatusSubmissionPending}
	assert.NoError(t, repo.CreateSubmission(ctx, s))

	got, err := repo.GetSubmissionByEnrollmentAssignment(ctx, 20, 10)
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, s.ID, got.ID)
}
