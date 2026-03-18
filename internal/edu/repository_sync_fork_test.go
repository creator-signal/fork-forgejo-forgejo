package edu

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateSyncForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	task := &SyncForkTask{
		AssignmentID: 1,
		CreatorID:    50,
		TotalRepos:   5,
		Status:       "running",
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	mock.ExpectQuery(`INSERT INTO sync_fork_task`).
		WithArgs(task.AssignmentID, task.CreatorID, task.TotalRepos, task.Synced, task.Skipped, task.Failed, task.Status, task.ErrorLog, task.CreatedUnix, task.UpdatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err = repo.CreateSyncForkTask(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), task.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSyncForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM sync_fork_task WHERE`).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "creator_id", "total_repos", "synced", "skipped", "failed", "status", "error_log", "created_unix", "updated_unix"}).
			AddRow(1, 10, 50, 5, 3, 1, 1, "error", "repo1: err\n", now, now))

	task, err := repo.GetSyncForkTask(context.Background(), 1)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, int64(1), task.ID)
	assert.Equal(t, 5, task.TotalRepos)
	assert.Equal(t, 3, task.Synced)
	assert.Equal(t, 1, task.Skipped)
	assert.Equal(t, 1, task.Failed)
	assert.Equal(t, "error", task.Status)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetSyncForkTaskByAssignment(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	mock.ExpectQuery(`SELECT .+ FROM sync_fork_task WHERE .+ ORDER BY created_unix DESC LIMIT 1`).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "assignment_id", "creator_id", "total_repos", "synced", "skipped", "failed", "status", "error_log", "created_unix", "updated_unix"}).
			AddRow(2, 10, 50, 10, 8, 2, 0, "done", "", now, now))

	task, err := repo.GetSyncForkTaskByAssignment(context.Background(), 10)
	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, int64(2), task.ID)
	assert.Equal(t, "done", task.Status)
	assert.Equal(t, 8, task.Synced)
	assert.Equal(t, 2, task.Skipped)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateSyncForkTask(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)

	now := time.Now().Unix()
	task := &SyncForkTask{
		ID:          1,
		Status:      "done",
		Synced:      5,
		Skipped:     0,
		Failed:      0,
		ErrorLog:    "",
		UpdatedUnix: now,
	}

	mock.ExpectExec(`UPDATE sync_fork_task SET`).
		WithArgs(task.Status, task.Synced, task.Skipped, task.Failed, task.ErrorLog, task.UpdatedUnix, task.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateSyncForkTask(context.Background(), task)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
