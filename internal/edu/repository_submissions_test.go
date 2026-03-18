package edu

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateSubmission_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()
	now := time.Now().Unix()

	sub := &Submission{
		AssignmentID:  1,
		UserID:        2,
		StudentRepoID: 3,
		Status:        "started",
		CreatedUnix:   now,
		UpdatedUnix:   now,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO edu_submissions (assignment_id,user_id,student_repo_id,status,created_unix,updated_unix) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`)).
		WithArgs(sub.AssignmentID, sub.UserID, sub.StudentRepoID, sub.Status, sub.CreatedUnix, sub.UpdatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(10))

	err = repo.CreateSubmission(ctx, sub)
	assert.NoError(t, err)
	assert.Equal(t, int64(10), sub.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubmission_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := &Submission{
		ID:            10,
		AssignmentID:  1,
		UserID:        2,
		StudentRepoID: 3,
		Status:        "started",
		Grade:         -1,
		Comment:       "",
		GradedByID:    0,
		GradedUnix:    0,
		CreatedUnix:   12345,
		UpdatedUnix:   12345,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, assignment_id, user_id, student_repo_id, status, grade, comment, graded_by_id, graded_unix, created_unix, updated_unix FROM edu_submissions WHERE assignment_id = $1 AND user_id = $2`)).
		WithArgs(expected.AssignmentID, expected.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "user_id", "student_repo_id", "status", "grade", "comment", "graded_by_id", "graded_unix", "created_unix", "updated_unix"}).
			AddRow(expected.ID, expected.AssignmentID, expected.UserID, expected.StudentRepoID, expected.Status, expected.Grade, expected.Comment, expected.GradedByID, expected.GradedUnix, expected.CreatedUnix, expected.UpdatedUnix))

	result, err := repo.GetSubmission(ctx, expected.AssignmentID, expected.UserID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSubmissionByRepoID_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := &Submission{
		ID:            10,
		AssignmentID:  1,
		UserID:        2,
		StudentRepoID: 3,
		Status:        "started",
		Grade:         -1,
		Comment:       "",
		GradedByID:    0,
		GradedUnix:    0,
		CreatedUnix:   12345,
		UpdatedUnix:   12345,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, assignment_id, user_id, student_repo_id, status, grade, comment, graded_by_id, graded_unix, created_unix, updated_unix FROM edu_submissions WHERE student_repo_id = $1`)).
		WithArgs(expected.StudentRepoID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "user_id", "student_repo_id", "status", "grade", "comment", "graded_by_id", "graded_unix", "created_unix", "updated_unix"}).
			AddRow(expected.ID, expected.AssignmentID, expected.UserID, expected.StudentRepoID, expected.Status, expected.Grade, expected.Comment, expected.GradedByID, expected.GradedUnix, expected.CreatedUnix, expected.UpdatedUnix))

	result, err := repo.GetSubmissionByRepoID(ctx, expected.StudentRepoID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSubmission_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	sub := &Submission{
		ID:          10,
		Status:      "passed",
		UpdatedUnix: 99999,
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE edu_submissions SET status = $1, updated_unix = $2 WHERE id = $3`)).
		WithArgs(sub.Status, sub.UpdatedUnix, sub.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateSubmission(ctx, sub)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
