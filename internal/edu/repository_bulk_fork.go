package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateBulkForkTask(ctx context.Context, task *BulkForkTask) error {
	_, err := db.GetEngine(ctx).Insert(task)
	if err != nil {
		return fmt.Errorf("insert bulk fork task: %w", err)
	}
	return nil
}

func (r *xormRepository) GetBulkForkTask(ctx context.Context, id int64) (*BulkForkTask, error) {
	t := &BulkForkTask{}
	has, err := db.GetEngine(ctx).ID(id).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get bulk fork task: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) GetBulkForkTaskByAssignment(ctx context.Context, assignmentID int64) (*BulkForkTask, error) {
	t := &BulkForkTask{}
	has, err := db.GetEngine(ctx).Where("assignment_id = ?", assignmentID).OrderBy("created_unix DESC").Limit(1).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get bulk fork task by assignment: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) UpdateBulkForkTask(ctx context.Context, task *BulkForkTask) error {
	_, err := db.GetEngine(ctx).ID(task.ID).Cols("status", "completed", "failed", "error_log", "updated_unix").Update(task)
	if err != nil {
		return fmt.Errorf("update bulk fork task: %w", err)
	}
	return nil
}
