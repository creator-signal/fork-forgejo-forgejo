// Copyright 2022 The Gitea Authors. All rights reserved.
// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/container"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/timeutil"

	"xorm.io/builder"
)

type TaskList []*ActionTask

func (tasks TaskList) GetJobIDs() []int64 {
	return container.FilterSlice(tasks, func(t *ActionTask) (int64, bool) {
		return t.JobID, t.JobID != 0
	})
}

func (tasks TaskList) LoadJobs(ctx context.Context) error {
	jobIDs := tasks.GetJobIDs()
	jobs := make(map[int64]*ActionRunJob, len(jobIDs))
	if err := db.GetEngine(ctx).In("id", jobIDs).Find(&jobs); err != nil {
		return err
	}
	for _, t := range tasks {
		if t.JobID > 0 && t.Job == nil {
			t.Job = jobs[t.JobID]
		}
	}

	// TODO: Replace with "ActionJobList(maps.Values(jobs))" once available
	var jobsList ActionJobList = make([]*ActionRunJob, 0, len(jobs))
	for _, j := range jobs {
		jobsList = append(jobsList, j)
	}
	return jobsList.LoadAttributes(ctx, true)
}

func (tasks TaskList) LoadAttributes(ctx context.Context) error {
	if err := tasks.LoadJobs(ctx); err != nil {
		return err
	}
	return tasks.LoadSteps(ctx)
}

// LoadSteps loads the Steps of every task in the list with a single batched
// query instead of one query per task.
func (tasks TaskList) LoadSteps(ctx context.Context) error {
	taskIDs := make([]int64, 0, len(tasks))
	for _, t := range tasks {
		if t.Steps == nil { // be careful, an empty slice (not nil) also means loaded
			taskIDs = append(taskIDs, t.ID)
		}
	}
	if len(taskIDs) == 0 {
		return nil
	}

	var steps []*ActionTaskStep
	if err := db.GetEngine(ctx).In("task_id", taskIDs).OrderBy("task_id ASC, `index` ASC").Find(&steps); err != nil {
		return err
	}

	stepsByTask := make(map[int64][]*ActionTaskStep, len(taskIDs))
	for _, s := range steps {
		stepsByTask[s.TaskID] = append(stepsByTask[s.TaskID], s)
	}
	for _, t := range tasks {
		if t.Steps == nil {
			if s, ok := stepsByTask[t.ID]; ok {
				t.Steps = s
			} else {
				t.Steps = []*ActionTaskStep{}
			}
		}
	}
	return nil
}

type FindTaskOptions struct {
	db.ListOptions
	RepoID        int64
	OwnerID       int64
	CommitSHA     string
	Status        []Status
	UpdatedBefore timeutil.TimeStamp
	StartedBefore timeutil.TimeStamp
	RunnerID      int64
	LogExpired    optional.Option[bool]
	LogInStorage  optional.Option[bool]
}

func (opts FindTaskOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.RepoID > 0 {
		cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	}
	if opts.OwnerID != 0 {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}
	if opts.CommitSHA != "" {
		cond = cond.And(builder.Eq{"commit_sha": opts.CommitSHA})
	}
	if len(opts.Status) > 0 {
		cond = cond.And(builder.In("status", opts.Status))
	}
	if opts.UpdatedBefore > 0 {
		cond = cond.And(builder.Lt{"updated": opts.UpdatedBefore})
	}
	if opts.StartedBefore > 0 {
		cond = cond.And(builder.Lt{"started": opts.StartedBefore})
	}
	if opts.RunnerID > 0 {
		cond = cond.And(builder.Eq{"runner_id": opts.RunnerID})
	}
	if has, value := opts.LogExpired.Get(); has {
		cond = cond.And(builder.Eq{"log_expired": value})
	}
	if has, value := opts.LogInStorage.Get(); has {
		cond = cond.And(builder.Eq{"log_in_storage": value})
	}
	return cond
}

func (opts FindTaskOptions) ToOrders() string {
	return "`id` DESC"
}
