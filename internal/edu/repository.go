package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

// dbRepository is a SQL implementation of Repository.
type dbRepository struct {
	runner SQLRunner
}

// SQLRunner interface abstracts the database connection (sql.DB or sql.Tx).
type SQLRunner interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

// NewRepository creates a new instance of Repository.
func NewRepository(runner SQLRunner) Repository {
	return &dbRepository{runner: runner}
}

func (r *dbRepository) CreateAssignment(ctx context.Context, a *Assignment) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_assignments").
		Columns("course_id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix").
		Values(a.CourseID, a.RepoID, a.Title, a.Description, a.DeadlineUnix, a.CreatedUnix, a.UpdatedUnix).
		Suffix("RETURNING id").
		ToSql()

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&a.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetSubmissions(ctx context.Context, assignmentID int64) ([]*Submission, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "user_id", "student_repo_id", "status", "grade", "comment", "graded_by_id", "graded_unix", "created_unix", "updated_unix").
		From("edu_submissions").
		Where(sq.Eq{"assignment_id": assignmentID}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var submissions []*Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(&s.ID, &s.AssignmentID, &s.UserID, &s.StudentRepoID, &s.Status, &s.Grade, &s.Comment, &s.GradedByID, &s.GradedUnix, &s.CreatedUnix, &s.UpdatedUnix)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		submissions = append(submissions, &s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return submissions, nil
}

func (r *dbRepository) GetAssignmentByID(ctx context.Context, id int64) (*Assignment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix").
		From("edu_assignments").
		Where(sq.Eq{"id": id}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var a Assignment
	err = row.Scan(&a.ID, &a.CourseID, &a.RepoID, &a.Title, &a.Description, &a.DeadlineUnix, &a.CreatedUnix, &a.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &a, nil
}
