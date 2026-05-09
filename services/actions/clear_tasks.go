// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"fmt"
	"time"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
)

// StopZombieTasks stops the task which have running status, but haven't been updated for a long time
//
// "Haven't been updated" alone is *not* enough evidence that a task is dead — under load the
// runner's UpdateTask RPC for one specific task can stall for many minutes (large log payloads
// piling up in the per-task reporter, the single clientM mutex serialising the whole pipeline,
// or a hung TCP socket without an explicit per-call timeout) while the runner itself is still
// happily executing the build and pinging the server through Ping/FetchTask. Killing such a
// task here turns into the symptom we hit on long Qt/Chromium-style builds: the runner reports
// SUCCESS at the end, but the server has already finalised the row as FAILURE and the runner
// receives the bogus result back via UpdateTaskResponse.State.
//
// To avoid that false positive we cross-check the assigned runner's LastOnline. If the runner
// itself was online within ZombieTaskTimeout, treat the task as still in progress and let the
// reporter eventually complete or the EndlessTask reaper handle the truly endless case.
func StopZombieTasks(ctx context.Context) error {
	threshold := timeutil.TimeStamp(time.Now().Add(-setting.Actions.ZombieTaskTimeout).Unix())
	tasks, err := db.Find[actions_model.ActionTask](ctx, actions_model.FindTaskOptions{
		Status:        []actions_model.Status{actions_model.StatusRunning},
		UpdatedBefore: threshold,
	})
	if err != nil {
		return fmt.Errorf("find zombie task candidates: %w", err)
	}

	live := make([]*actions_model.ActionTask, 0, len(tasks))
	for _, task := range tasks {
		runner := &actions_model.ActionRunner{}
		has, err := db.GetEngine(ctx).ID(task.RunnerID).Get(runner)
		if err != nil {
			log.Warn("Cannot load runner %d for zombie task %d: %v", task.RunnerID, task.ID, err)
		}
		// No runner row, or the runner has not been seen since the zombie threshold:
		// treat as a real zombie and add it to the kill list.
		if !has || runner.LastOnline < threshold {
			live = append(live, task)
		}
	}

	return stopTasksList(ctx, live)
}

// StopEndlessTasks stops the tasks which have running status and continuous updates, but don't end for a long time
func StopEndlessTasks(ctx context.Context) error {
	return stopTasks(ctx, actions_model.FindTaskOptions{
		Status:        []actions_model.Status{actions_model.StatusRunning},
		StartedBefore: timeutil.TimeStamp(time.Now().Add(-setting.Actions.EndlessTaskTimeout).Unix()),
	})
}

func stopTasks(ctx context.Context, opts actions_model.FindTaskOptions) error {
	tasks, err := db.Find[actions_model.ActionTask](ctx, opts)
	if err != nil {
		return fmt.Errorf("find tasks: %w", err)
	}
	return stopTasksList(ctx, tasks)
}

// stopTasksList does the per-task termination work for an already-resolved
// slice of tasks. Pulled out of stopTasks so StopZombieTasks can pre-filter
// the candidate set with a runner-liveness check (see comment on StopZombieTasks).
func stopTasksList(ctx context.Context, tasks []*actions_model.ActionTask) error {
	jobs := make([]*actions_model.ActionRunJob, 0, len(tasks))
	for _, task := range tasks {
		if err := db.WithTx(ctx, func(ctx context.Context) error {
			if err := StopTask(ctx, task.ID, actions_model.StatusFailure); err != nil {
				return err
			}
			if err := task.LoadJob(ctx); err != nil {
				return err
			}
			jobs = append(jobs, task.Job)
			return nil
		}); err != nil {
			log.Warn("Cannot stop task %v: %v", task.ID, err)
			continue
		}

		remove, err := actions.TransferLogs(ctx, task.LogFilename)
		if err != nil {
			log.Warn("Cannot transfer logs of task %v: %v", task.ID, err)
			continue
		}
		task.LogInStorage = true
		if err := actions_model.UpdateTask(ctx, task, "log_in_storage"); err != nil {
			log.Warn("Cannot update task %v: %v", task.ID, err)
			continue
		}
		remove()
	}

	CreateCommitStatus(ctx, jobs...)

	return nil
}

// CancelAbandonedJobs cancels the jobs which have waiting status, but haven't been picked by a runner for a long time
func CancelAbandonedJobs(ctx context.Context) error {
	jobs, err := db.Find[actions_model.ActionRunJob](ctx, actions_model.FindRunJobOptions{
		Statuses:         []actions_model.Status{actions_model.StatusWaiting, actions_model.StatusBlocked},
		UpdatedBefore:    timeutil.TimeStamp(time.Now().Add(-setting.Actions.AbandonedJobTimeout).Unix()),
		RunNeedsApproval: optional.Some(false),
	})
	if err != nil {
		log.Warn("find abandoned tasks: %v", err)
		return err
	}

	now := timeutil.TimeStampNow()
	for _, job := range jobs {
		job.Status = actions_model.StatusCancelled
		job.Stopped = now
		if err := db.WithTx(ctx, func(ctx context.Context) error {
			_, err := UpdateRunJob(ctx, job, nil, "status", "stopped")
			return err
		}); err != nil {
			log.Warn("cancel abandoned job %v: %v", job.ID, err)
			// go on
		}
		CreateCommitStatus(ctx, job)
	}

	return nil
}
