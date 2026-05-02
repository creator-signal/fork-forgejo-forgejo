package edu

import (
	"context"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateCourseSyncPR(ctx context.Context, p *CourseSyncPR) error {
	_, err := db.GetEngine(ctx).Insert(p)
	return err
}

func (r *xormRepository) UpdateCourseSyncPR(ctx context.Context, p *CourseSyncPR) error {
	_, err := db.GetEngine(ctx).ID(p.ID).AllCols().Update(p)
	return err
}

func (r *xormRepository) ListCourseSyncPRsByTask(ctx context.Context, taskID int64) ([]*CourseSyncPR, error) {
	var list []*CourseSyncPR
	err := db.GetEngine(ctx).Where("sync_task_id = ?", taskID).Find(&list)
	return list, err
}

func (r *xormRepository) GetCourseSyncPR(ctx context.Context, id int64) (*CourseSyncPR, error) {
	p := &CourseSyncPR{}
	has, err := db.GetEngine(ctx).ID(id).Get(p)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return p, nil
}
