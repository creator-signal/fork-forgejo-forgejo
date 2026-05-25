// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"fmt"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"
)

// ActionTaskStep represents a step of ActionTask
type ActionTaskStep struct {
	ID        int64
	Name      string `xorm:"VARCHAR(255)"`
	TaskID    int64  `xorm:"index unique(task_index)"`
	Index     int64  `xorm:"index unique(task_index)"`
	RepoID    int64  `xorm:"index"`
	Status    Status `xorm:"index"`
	LogIndex  int64
	LogLength int64
	Started   timeutil.TimeStamp
	Stopped   timeutil.TimeStamp
	Created   timeutil.TimeStamp `xorm:"created"`
	Updated   timeutil.TimeStamp `xorm:"updated"`
}

func (step *ActionTaskStep) Duration() time.Duration {
	return calculateDuration(step.Started, step.Stopped, step.Status)
}

func init() {
	db.RegisterModel(new(ActionTaskStep))
}

func GetTaskStepsByTaskID(ctx context.Context, taskID int64) ([]*ActionTaskStep, error) {
	var steps []*ActionTaskStep
	return steps, db.GetEngine(ctx).Where("task_id=?", taskID).OrderBy("`index` ASC").Find(&steps)
}

func GetTaskStepByTaskIDAndIndex(ctx context.Context, taskID, index int64) (*ActionTaskStep, error) {
	var step ActionTaskStep
	has, err := db.GetEngine(ctx).Where("task_id=? AND `index`=?", taskID, index).Get(&step)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("task step with task_id %d and index %d: %w", taskID, index, util.ErrNotExist)
	}
	return &step, nil
}
