package edu

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateTestResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	tr := &TestResult{
		SubmissionID: 10,
		CommitSHA:    "abc1234def5678",
		Score:        85,
		Details:      "5/6 tests passed",
		CreatedUnix:  now,
	}

	mock.ExpectQuery(`INSERT INTO edu_test_results`).
		WithArgs(tr.SubmissionID, tr.CommitSHA, tr.Score, tr.Details, tr.CreatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err = repo.CreateTestResult(context.Background(), tr)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), tr.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetTestResultsBySubmission(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM edu_test_results WHERE .+ ORDER BY created_unix DESC`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_id", "commit_sha", "score", "details", "created_unix"}).
			AddRow(2, 10, "def5678", 100, "All passed", now).
			AddRow(1, 10, "abc1234", 50, "3/6 passed", now-100))

	results, err := repo.GetTestResultsBySubmission(context.Background(), 10)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, int64(2), results[0].ID)
	assert.Equal(t, 100, results[0].Score)
	assert.Equal(t, int64(1), results[1].ID)
	assert.Equal(t, 50, results[1].Score)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetTestResultsBySubmission_Empty(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM edu_test_results WHERE`).
		WithArgs(int64(99)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_id", "commit_sha", "score", "details", "created_unix"}))

	results, err := repo.GetTestResultsBySubmission(context.Background(), 99)
	assert.NoError(t, err)
	assert.Empty(t, results)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetLatestTestResult(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM edu_test_results WHERE .+ ORDER BY created_unix DESC LIMIT 1`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "submission_id", "commit_sha", "score", "details", "created_unix"}).
			AddRow(5, 10, "abc1234", 100, "All passed", now))

	tr, err := repo.GetLatestTestResult(context.Background(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, tr)
	assert.Equal(t, int64(5), tr.ID)
	assert.Equal(t, 100, tr.Score)
	assert.Equal(t, "abc1234", tr.CommitSHA)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepo_GetLatestTestResult_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	mock.ExpectQuery(`SELECT .+ FROM edu_test_results WHERE .+ ORDER BY created_unix DESC LIMIT 1`).
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	tr, err := repo.GetLatestTestResult(context.Background(), 99)
	assert.NoError(t, err)
	assert.Nil(t, tr)
	assert.NoError(t, mock.ExpectationsWereMet())
}
