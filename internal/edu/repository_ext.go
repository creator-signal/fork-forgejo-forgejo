package edu

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix").
		From("edu_assignments").
		Where(sq.Eq{"repo_id": repoID}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.CourseID, &a.RepoID, &a.Title, &a.Description, &a.DeadlineUnix, &a.CreatedUnix, &a.UpdatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		assignments = append(assignments, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return assignments, nil
}

func (r *dbRepository) GetAssignmentsForUser(ctx context.Context, userID int64) ([]*Assignment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.
		Select("a.id", "a.course_id", "a.repo_id", "a.title", "a.description", "a.deadline_unix", "a.created_unix", "a.updated_unix").
		From("edu_assignments a").
		Join("edu_course_enrollments e ON a.course_id = e.course_id").
		Where(sq.Eq{"e.user_id": userID}).
		OrderBy("a.created_unix DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.CourseID, &a.RepoID, &a.Title, &a.Description, &a.DeadlineUnix, &a.CreatedUnix, &a.UpdatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		assignments = append(assignments, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return assignments, nil
}

func (r *dbRepository) UpdateAssignment(ctx context.Context, a *Assignment) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("edu_assignments").
		Set("title", a.Title).
		Set("description", a.Description).
		Set("deadline_unix", a.DeadlineUnix).
		Set("updated_unix", a.UpdatedUnix).
		Where(sq.Eq{"id": a.ID}).
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

func (r *dbRepository) DeleteAssignment(ctx context.Context, id int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// Delete related submissions first
	query, args, err := psql.Delete("edu_submissions").
		Where(sq.Eq{"assignment_id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete submissions query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete submissions: %w", err)
	}

	// Delete the assignment
	query, args, err = psql.Delete("edu_assignments").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete assignment query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete assignment: %w", err)
	}

	return nil
}
