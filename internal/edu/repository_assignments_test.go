package edu

import (
	"context"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetAssignmentsForUser_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT a.id, a.course_id, a.repo_id, a.title, a.description, a.deadline_unix, a.created_unix, a.updated_unix FROM edu_assignments a JOIN edu_course_enrollments e ON a.course_id = e.course_id WHERE e.user_id = $1 ORDER BY a.created_unix DESC`)).
		WithArgs(int64(5)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "course_id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix"}).
			AddRow(1, 10, 100, "HW1", "Do stuff", 0, 1000, 1000).
			AddRow(2, 10, 101, "HW2", "More stuff", 9999, 1001, 1001))

	result, err := repo.GetAssignmentsForUser(ctx, 5)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "HW1", result[0].Title)
	assert.Equal(t, int64(10), result[0].CourseID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateAssignment_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	a := &Assignment{
		ID:           1,
		Title:        "Updated HW",
		Description:  "New desc",
		DeadlineUnix: 5000,
		UpdatedUnix:  6000,
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE edu_assignments SET title = $1, description = $2, deadline_unix = $3, updated_unix = $4 WHERE id = $5`)).
		WithArgs(a.Title, a.Description, a.DeadlineUnix, a.UpdatedUnix, a.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateAssignment(ctx, a)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAssignment_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	// Expect delete submissions first
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM edu_submissions WHERE assignment_id = $1`)).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	// Then delete assignment
	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM edu_assignments WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteAssignment(ctx, 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
