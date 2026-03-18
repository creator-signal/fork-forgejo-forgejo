package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) CreateCourse(ctx context.Context, c *Course) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_courses").
		Columns("name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix").
		Values(c.Name, c.Description, c.CreatorID, c.OrgID, c.StartUnix, c.EndUnix, c.CreatedUnix, c.UpdatedUnix).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&c.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetCourseByID(ctx context.Context, id int64) (*Course, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix").
		From("edu_courses").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var c Course
	err = row.Scan(&c.ID, &c.Name, &c.Description, &c.CreatorID, &c.OrgID, &c.StartUnix, &c.EndUnix, &c.CreatedUnix, &c.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &c, nil
}

func (r *dbRepository) GetCoursesByCreator(ctx context.Context, creatorID int64) ([]*Course, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "name", "description", "creator_id", "org_id", "start_unix", "end_unix", "created_unix", "updated_unix").
		From("edu_courses").
		Where(sq.Eq{"creator_id": creatorID}).
		OrderBy("created_unix DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var courses []*Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatorID, &c.OrgID, &c.StartUnix, &c.EndUnix, &c.CreatedUnix, &c.UpdatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		courses = append(courses, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return courses, nil
}

func (r *dbRepository) GetCoursesByUser(ctx context.Context, userID int64) ([]*Course, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("c.id", "c.name", "c.description", "c.creator_id", "c.org_id", "c.start_unix", "c.end_unix", "c.created_unix", "c.updated_unix").
		From("edu_courses c").
		Join("edu_course_enrollments e ON e.course_id = c.id").
		Where(sq.Eq{"e.user_id": userID}).
		OrderBy("c.created_unix DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var courses []*Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.Name, &c.Description, &c.CreatorID, &c.OrgID, &c.StartUnix, &c.EndUnix, &c.CreatedUnix, &c.UpdatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		courses = append(courses, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return courses, nil
}

func (r *dbRepository) UpdateCourse(ctx context.Context, c *Course) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("edu_courses").
		Set("name", c.Name).
		Set("description", c.Description).
		Set("start_unix", c.StartUnix).
		Set("end_unix", c.EndUnix).
		Set("updated_unix", c.UpdatedUnix).
		Where(sq.Eq{"id": c.ID}).
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

func (r *dbRepository) DeleteCourse(ctx context.Context, id int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Delete("edu_courses").
		Where(sq.Eq{"id": id}).
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

func (r *dbRepository) GetAssignmentsByCourse(ctx context.Context, courseID int64) ([]*Assignment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix").
		From("edu_assignments").
		Where(sq.Eq{"course_id": courseID}).
		OrderBy("created_unix DESC").
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
