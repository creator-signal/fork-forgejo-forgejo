package edu

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateCourse_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now().Unix()
	c := &Course{
		Name:        "Software Engineering Spring 2026",
		Description: "Intro course",
		CreatorID:   1,
		OrgID:       0,
		StartUnix:   now,
		EndUnix:     now + 86400*90,
		CreatedUnix: now,
		UpdatedUnix: now,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO edu_courses (name,description,creator_id,org_id,start_unix,end_unix,created_unix,updated_unix) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`)).
		WithArgs(c.Name, c.Description, c.CreatorID, c.OrgID, c.StartUnix, c.EndUnix, c.CreatedUnix, c.UpdatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err = repo.CreateCourse(ctx, c)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), c.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCourseByID_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	expected := &Course{
		ID:          1,
		Name:        "Test Course",
		Description: "Desc",
		CreatorID:   10,
		OrgID:       0,
		StartUnix:   1000,
		EndUnix:     2000,
		CreatedUnix: 1000,
		UpdatedUnix: 1000,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, creator_id, org_id, start_unix, end_unix, created_unix, updated_unix FROM edu_courses WHERE id = $1`)).
		WithArgs(expected.ID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix"}).
			AddRow(expected.ID, expected.Name, expected.Description, expected.CreatorID, expected.OrgID, expected.StartUnix, expected.EndUnix, expected.CreatedUnix, expected.UpdatedUnix))

	result, err := repo.GetCourseByID(ctx, expected.ID)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCourseByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, creator_id, org_id, start_unix, end_unix, created_unix, updated_unix FROM edu_courses WHERE id = $1`)).
		WithArgs(int64(999)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix"}))

	result, err := repo.GetCourseByID(ctx, 999)
	assert.NoError(t, err)
	assert.Nil(t, result)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCoursesByCreator_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, name, description, creator_id, org_id, start_unix, end_unix, created_unix, updated_unix FROM edu_courses WHERE creator_id = $1 ORDER BY created_unix DESC`)).
		WithArgs(int64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix"}).
			AddRow(1, "Course 1", "Desc 1", 10, 0, 1000, 2000, 1000, 1000).
			AddRow(2, "Course 2", "Desc 2", 10, 0, 1000, 2000, 900, 900))

	result, err := repo.GetCoursesByCreator(ctx, 10)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "Course 1", result[0].Name)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateCourse_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	c := &Course{
		ID:          1,
		Name:        "Updated",
		Description: "New desc",
		StartUnix:   1000,
		EndUnix:     2000,
		UpdatedUnix: 3000,
	}

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE edu_courses SET name = $1, description = $2, start_unix = $3, end_unix = $4, updated_unix = $5 WHERE id = $6`)).
		WithArgs(c.Name, c.Description, c.StartUnix, c.EndUnix, c.UpdatedUnix, c.ID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateCourse(ctx, c)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteCourse_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM edu_courses WHERE id = $1`)).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.DeleteCourse(ctx, 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEnrollUser_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	now := time.Now().Unix()
	e := &CourseEnrollment{
		CourseID:    1,
		UserID:      5,
		Role:        RoleStudent,
		CreatedUnix: now,
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO edu_course_enrollments (course_id,user_id,role,created_unix) VALUES ($1,$2,$3,$4) RETURNING id`)).
		WithArgs(e.CourseID, e.UserID, e.Role, e.CreatedUnix).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	err = repo.EnrollUser(ctx, e)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), e.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetEnrollments_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, course_id, user_id, role, created_unix FROM edu_course_enrollments WHERE course_id = $1 ORDER BY created_unix ASC`)).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "course_id", "user_id", "role", "created_unix"}).
			AddRow(1, 1, 5, "student", 1000).
			AddRow(2, 1, 6, "teacher", 1001))

	result, err := repo.GetEnrollments(ctx, 1)
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, RoleStudent, result[0].Role)
	assert.Equal(t, RoleTeacher, result[1].Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRemoveEnrollment_Repo(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM edu_course_enrollments WHERE course_id = $1 AND user_id = $2`)).
		WithArgs(int64(1), int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.RemoveEnrollment(ctx, 1, 5)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
