package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) EnrollUser(ctx context.Context, enrollment *CourseEnrollment) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_course_enrollments").
		Columns("course_id", "user_id", "role", "created_unix").
		Values(enrollment.CourseID, enrollment.UserID, enrollment.Role, enrollment.CreatedUnix).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&enrollment.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetEnrollment(ctx context.Context, courseID, userID int64) (*CourseEnrollment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "user_id", "role", "created_unix").
		From("edu_course_enrollments").
		Where(sq.Eq{"course_id": courseID, "user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var e CourseEnrollment
	err = row.Scan(&e.ID, &e.CourseID, &e.UserID, &e.Role, &e.CreatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &e, nil
}

func (r *dbRepository) GetEnrollments(ctx context.Context, courseID int64) ([]*CourseEnrollment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "user_id", "role", "created_unix").
		From("edu_course_enrollments").
		Where(sq.Eq{"course_id": courseID}).
		OrderBy("created_unix ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var enrollments []*CourseEnrollment
	for rows.Next() {
		var e CourseEnrollment
		if err := rows.Scan(&e.ID, &e.CourseID, &e.UserID, &e.Role, &e.CreatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		enrollments = append(enrollments, &e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return enrollments, nil
}

func (r *dbRepository) RemoveEnrollment(ctx context.Context, courseID, userID int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Delete("edu_course_enrollments").
		Where(sq.Eq{"course_id": courseID, "user_id": userID}).
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
