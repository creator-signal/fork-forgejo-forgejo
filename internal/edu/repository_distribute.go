package edu

import (
	"context"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateDistributeTask(ctx context.Context, t *DistributeTask) error {
	_, err := db.GetEngine(ctx).Insert(t)
	return err
}

func (r *xormRepository) GetDistributeTask(ctx context.Context, id int64) (*DistributeTask, error) {
	t := &DistributeTask{}
	has, err := db.GetEngine(ctx).ID(id).Get(t)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return t, nil
}

func (r *xormRepository) UpdateDistributeTask(ctx context.Context, t *DistributeTask) error {
	_, err := db.GetEngine(ctx).ID(t.ID).AllCols().Update(t)
	return err
}

func (r *xormRepository) GetDistributeTaskByAssignment(ctx context.Context, assignmentID int64) (*DistributeTask, error) {
	t := &DistributeTask{}
	has, err := db.GetEngine(ctx).Where("assignment_id = ?", assignmentID).Desc("id").Get(t)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, nil
	}
	return t, nil
}
