package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateInitForksTask(ctx context.Context, task *InitForksTask) error {
	_, err := db.GetEngine(ctx).Insert(task)
	if err != nil {
		return fmt.Errorf("insert init forks task: %w", err)
	}
	return nil
}

func (r *xormRepository) GetInitForksTask(ctx context.Context, id int64) (*InitForksTask, error) {
	t := &InitForksTask{}
	has, err := db.GetEngine(ctx).ID(id).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get init forks task: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) GetInitForksTaskByCourse(ctx context.Context, courseID int64) (*InitForksTask, error) {
	t := &InitForksTask{}
	has, err := db.GetEngine(ctx).Where("course_id = ?", courseID).OrderBy("created_unix DESC").Limit(1).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get init forks task by course: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) UpdateInitForksTask(ctx context.Context, task *InitForksTask) error {
	_, err := db.GetEngine(ctx).ID(task.ID).Cols("status", "completed", "failed", "error_log", "updated_unix").Update(task)
	if err != nil {
		return fmt.Errorf("update init forks task: %w", err)
	}
	return nil
}
