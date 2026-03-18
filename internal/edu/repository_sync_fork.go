package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) CreateSyncForkTask(ctx context.Context, task *SyncForkTask) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("sync_fork_task").
		Columns("assignment_id", "creator_id", "total_repos", "synced", "skipped", "failed", "status", "error_log", "created_unix", "updated_unix").
		Values(task.AssignmentID, task.CreatorID, task.TotalRepos, task.Synced, task.Skipped, task.Failed, task.Status, task.ErrorLog, task.CreatedUnix, task.UpdatedUnix).
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

func (r *dbRepository) GetSyncForkTask(ctx context.Context, id int64) (*SyncForkTask, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "creator_id", "total_repos", "synced", "skipped", "failed", "status", "error_log", "created_unix", "updated_unix").
		From("sync_fork_task").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var t SyncForkTask
	err = row.Scan(&t.ID, &t.AssignmentID, &t.CreatorID, &t.TotalRepos, &t.Synced, &t.Skipped, &t.Failed, &t.Status, &t.ErrorLog, &t.CreatedUnix, &t.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &t, nil
}

func (r *dbRepository) GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "assignment_id", "creator_id", "total_repos", "synced", "skipped", "failed", "status", "error_log", "created_unix", "updated_unix").
		From("sync_fork_task").
		Where(sq.Eq{"assignment_id": assignmentID}).
		OrderBy("created_unix DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var t SyncForkTask
	err = row.Scan(&t.ID, &t.AssignmentID, &t.CreatorID, &t.TotalRepos, &t.Synced, &t.Skipped, &t.Failed, &t.Status, &t.ErrorLog, &t.CreatedUnix, &t.UpdatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &t, nil
}

func (r *dbRepository) UpdateSyncForkTask(ctx context.Context, task *SyncForkTask) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("sync_fork_task").
		Set("status", task.Status).
		Set("synced", task.Synced).
		Set("skipped", task.Skipped).
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
