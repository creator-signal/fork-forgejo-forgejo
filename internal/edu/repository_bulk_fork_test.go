package edu

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateBulkForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	task := &BulkForkTask{
		AssignmentID: 1,
		CreatorID:    50,
		TotalUsers:   5,
		Status:       "running",
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	mock.ExpectQuery(`INSERT INTO bulk_fork_task`).
		WithArgs(task.AssignmentID, task.CreatorID, task.TotalUsers, task.Completed, task.Failed, task.Status, task.ErrorLog, task.CreatedUnix, task.UpdatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err = repo.CreateBulkForkTask(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), task.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBulkForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM bulk_fork_task WHERE`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "creator_id", "total_users", "completed", "failed", "status", "error_log", "created_unix", "updated_unix"}).
			AddRow(1, 10, 50, 5, 3, 1, "error", "user1: err\n", now, now))

	task, err := repo.GetBulkForkTask(context.Background(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, int64(1), task.ID)
	assert.Equal(t, 5, task.TotalUsers)
	assert.Equal(t, 3, task.Completed)
	assert.Equal(t, 1, task.Failed)
	assert.Equal(t, "error", task.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetBulkForkTaskByAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM bulk_fork_task WHERE .+ ORDER BY created_unix DESC LIMIT 1`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "creator_id", "total_users", "completed", "failed", "status", "error_log", "created_unix", "updated_unix"}).
			AddRow(2, 10, 50, 10, 10, 0, "done", "", now, now))

	task, err := repo.GetBulkForkTaskByAssignment(context.Background(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, int64(2), task.ID)
	assert.Equal(t, "done", task.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateBulkForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	task := &BulkForkTask{
		ID:          1,
		Status:      "done",
		Completed:   5,
		Failed:      0,
		ErrorLog:    "",
		UpdatedUnix: now,
	}

	mock.ExpectExec(`UPDATE bulk_fork_task SET`).
		WithArgs(task.Status, task.Completed, task.Failed, task.ErrorLog, task.UpdatedUnix, task.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateBulkForkTask(context.Background(), task)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
