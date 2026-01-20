package edu

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateAssignment_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now().Unix()
	a := &Assignment{
		RepoID:       10,
		Title:        "New Task",
		Description:  "Do something",
		DeadlineUnix: now + 3600,
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO edu_assignments (repo_id,title,description,deadline_unix,created_unix,updated_unix) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`)).
		WithArgs(a.RepoID, a.Title, a.Description, a.DeadlineUnix, a.CreatedUnix, a.UpdatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(100))

	err = repo.CreateAssignment(ctx, a)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), a.ID)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetAssignmentByID_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := &Assignment{
		ID:           100,
		RepoID:       10,
		Title:        "Existing Task",
		Description:  "Desc",
		DeadlineUnix: 12345,
		CreatedUnix:  12345,
		UpdatedUnix:  12345,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, repo_id, title, description, deadline_unix, created_unix, updated_unix FROM edu_assignments WHERE id = $1`)).
		WithArgs(expected.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix"}).
			AddRow(expected.ID, expected.RepoID, expected.Title, expected.Description, expected.DeadlineUnix, expected.CreatedUnix, expected.UpdatedUnix))

	result, err := repo.GetAssignmentByID(ctx, expected.ID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	assert.NoError(t, mock.ExpectationsWereMet())
}
