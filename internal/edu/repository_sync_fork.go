package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateSyncForkTask(ctx context.Context, task *SyncForkTask) error {
	_, err := db.GetEngine(ctx).Insert(task)
	if err != nil {
		return fmt.Errorf("insert sync fork task: %w", err)
	}
	return nil
}

func (r *xormRepository) GetSyncForkTask(ctx context.Context, id int64) (*SyncForkTask, error) {
	t := &SyncForkTask{}
	has, err := db.GetEngine(ctx).ID(id).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get sync fork task: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error) {
	t := &SyncForkTask{}
	has, err := db.GetEngine(ctx).Where("assignment_id = ?", assignmentID).OrderBy("created_unix DESC").Limit(1).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get sync fork task by assignment: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) UpdateSyncForkTask(ctx context.Context, task *SyncForkTask) error {
	_, err := db.GetEngine(ctx).ID(task.ID).Cols("status", "synced", "skipped", "failed", "error_log", "updated_unix").Update(task)
	if err != nil {
		return fmt.Errorf("update sync fork task: %w", err)
	}
	return nil
}
