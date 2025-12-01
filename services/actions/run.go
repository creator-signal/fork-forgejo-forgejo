// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

func killRun(ctx context.Context, run *actions_model.ActionRun, newStatus actions_model.Status) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			oldStatus := job.Status
			if oldStatus.IsDone() {
				continue
			}
			if job.TaskID == 0 {
				job.Status = newStatus
				job.Stopped = timeutil.TimeStampNow()
				_, err := actions_model.UpdateRunJobWithoutNotification(ctx, job, nil, "status", "stopped")
				if err != nil {
					return err
				}
				continue
			}
			if err := StopTask(ctx, job.TaskID, newStatus); err != nil {
				return err
			}
		}

		if run.NeedApproval {
			if err := actions_model.UpdateRunApprovalByID(ctx, run.ID, actions_model.DoesNotNeedApproval, 0); err != nil {
				return err
			}
		}

		CreateCommitStatus(ctx, jobs...)

		return nil
	})
}

func CancelRun(ctx context.Context, run *actions_model.ActionRun) error {
	return killRun(ctx, run, actions_model.StatusCancelled)
}

func ApproveRun(ctx context.Context, run *actions_model.ActionRun, doerID int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
		if err != nil {
			return err
		}
		for _, job := range jobs {
			if len(job.Needs) == 0 && job.Status.IsBlocked() {
				job.Status = actions_model.StatusWaiting
				_, err := UpdateRunJob(ctx, job, nil, "status")
				if err != nil {
					return err
				}
			}
		}
		CreateCommitStatus(ctx, jobs...)

		return actions_model.UpdateRunApprovalByID(ctx, run.ID, actions_model.DoesNotNeedApproval, doerID)
	})
}

func FailRunPreExecutionError(ctx context.Context, run *actions_model.ActionRun, errorCode actions_model.PreExecutionError, details []any) error {
	if run.PreExecutionErrorCode != 0 {
		// Already have one error; keep it.
		return nil
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		run.Status = actions_model.StatusFailure
		run.PreExecutionErrorCode = errorCode
		run.PreExecutionErrorDetails = details
		if err := actions_model.UpdateRunWithoutNotification(ctx, run,
			"pre_execution_error_code", "pre_execution_error_details", "status"); err != nil {
			return err
		}

		// Also mark every pending job as Failed so nothing remains in a waiting/blocked state.
		return killRun(ctx, run, actions_model.StatusFailure)
	})
}
