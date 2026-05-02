package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateCourseSyncTask(ctx context.Context, task *CourseSyncTask) error {
	_, err := db.GetEngine(ctx).Insert(task)
	if err != nil {
		return fmt.Errorf("insert course sync task: %w", err)
	}
	return nil
}

func (r *xormRepository) GetCourseSyncTask(ctx context.Context, id int64) (*CourseSyncTask, error) {
	t := &CourseSyncTask{}
	has, err := db.GetEngine(ctx).ID(id).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get course sync task: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) GetCourseSyncTaskByCourse(ctx context.Context, courseID int64) (*CourseSyncTask, error) {
	t := &CourseSyncTask{}
	has, err := db.GetEngine(ctx).Where("course_id = ?", courseID).OrderBy("created_unix DESC").Limit(1).Get(t)
	if err != nil {
		return nil, fmt.Errorf("get course sync task by course: %w", err)
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) UpdateCourseSyncTask(ctx context.Context, task *CourseSyncTask) error {
	_, err := db.GetEngine(ctx).ID(task.ID).Cols("status", "synced", "skipped", "failed", "error_log", "updated_unix").Update(task)
	if err != nil {
		return fmt.Errorf("update course sync task: %w", err)
	}
	return nil
}
