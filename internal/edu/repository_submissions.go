package edu

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
)

func timeNowUnix() int64 {
	return time.Now().Unix()
}

func (r *dbRepository) CreateSubmission(ctx context.Context, s *Submission) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_submissions").
		Columns("assignment_id", "user_id", "student_repo_id", "status", "created_unix", "updated_unix").
		Values(s.AssignmentID, s.UserID, s.StudentRepoID, s.Status, s.CreatedUnix, s.UpdatedUnix).
		Suffix("RETURNING id").
		ToSql()

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&s.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetSubmission(ctx context.Context, assignmentID, userID int64) (*Submission, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "user_id", "student_repo_id", "status", "grade", "comment", "graded_by_id", "graded_unix", "created_unix", "updated_unix").
		From("edu_submissions").
		Where(sq.Eq{"assignment_id": assignmentID, "user_id": userID}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var s Submission
	err = row.Scan(&s.ID, &s.AssignmentID, &s.UserID, &s.StudentRepoID, &s.Status, &s.Grade, &s.Comment, &s.GradedByID, &s.GradedUnix, &s.CreatedUnix, &s.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &s, nil
}

func (r *dbRepository) GetSubmissionByRepoID(ctx context.Context, repoID int64) (*Submission, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "user_id", "student_repo_id", "status", "grade", "comment", "graded_by_id", "graded_unix", "created_unix", "updated_unix").
		From("edu_submissions").
		Where(sq.Eq{"student_repo_id": repoID}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var s Submission
	err = row.Scan(&s.ID, &s.AssignmentID, &s.UserID, &s.StudentRepoID, &s.Status, &s.Grade, &s.Comment, &s.GradedByID, &s.GradedUnix, &s.CreatedUnix, &s.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &s, nil
}

func (r *dbRepository) UpdateSubmission(ctx context.Context, s *Submission) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("edu_submissions").
		Set("status", s.Status).
		Set("updated_unix", s.UpdatedUnix).
		Where(sq.Eq{"id": s.ID}).
		ToSql()

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GradeSubmission(ctx context.Context, submissionID int64, grade int, comment string, gradedByID int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	now := timeNowUnix()
	query, args, err := psql.Update("edu_submissions").
		Set("grade", grade).
		Set("comment", comment).
		Set("graded_by_id", gradedByID).
		Set("graded_unix", now).
		Set("status", "graded").
		Set("updated_unix", now).
		Where(sq.Eq{"id": submissionID}).
		ToSql()

	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}
