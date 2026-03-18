package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) CreateBulkForkTask(ctx context.Context, task *BulkForkTask) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("bulk_fork_task").
		Columns("assignment_id", "creator_id", "total_users", "completed", "failed", "status", "error_log", "created_unix", "updated_unix").
		Values(task.AssignmentID, task.CreatorID, task.TotalUsers, task.Completed, task.Failed, task.Status, task.ErrorLog, task.CreatedUnix, task.UpdatedUnix).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&task.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetBulkForkTask(ctx context.Context, id int64) (*BulkForkTask, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "creator_id", "total_users", "completed", "failed", "status", "error_log", "created_unix", "updated_unix").
		From("bulk_fork_task").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var t BulkForkTask
	err = row.Scan(&t.ID, &t.AssignmentID, &t.CreatorID, &t.TotalUsers, &t.Completed, &t.Failed, &t.Status, &t.ErrorLog, &t.CreatedUnix, &t.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &t, nil
}

func (r *dbRepository) GetBulkForkTaskByAssignment(ctx context.Context, assignmentID int64) (*BulkForkTask, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "creator_id", "total_users", "completed", "failed", "status", "error_log", "created_unix", "updated_unix").
		From("bulk_fork_task").
		Where(sq.Eq{"assignment_id": assignmentID}).
		OrderBy("created_unix DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var t BulkForkTask
	err = row.Scan(&t.ID, &t.AssignmentID, &t.CreatorID, &t.TotalUsers, &t.Completed, &t.Failed, &t.Status, &t.ErrorLog, &t.CreatedUnix, &t.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &t, nil
}

func (r *dbRepository) UpdateBulkForkTask(ctx context.Context, task *BulkForkTask) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("bulk_fork_task").
		Set("status", task.Status).
		Set("completed", task.Completed).
		Set("failed", task.Failed).
		Set("error_log", task.ErrorLog).
		Set("updated_unix", task.UpdatedUnix).
		Where(sq.Eq{"id": task.ID}).
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
