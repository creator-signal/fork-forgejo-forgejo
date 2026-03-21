// Copyright 2024 The Gitea Authors. All rights reserved.
// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/modules/container"

	"xorm.io/builder"
)

// GetAllRerunJobs get all jobs that need to be rerun when job should be rerun
func GetAllRerunJobs(job *actions_model.ActionRunJob, allJobs []*actions_model.ActionRunJob) []*actions_model.ActionRunJob {
	rerunJobs := []*actions_model.ActionRunJob{job}
	rerunJobsIDSet := make(container.Set[string])
	rerunJobsIDSet.Add(job.JobID)

	for {
		found := false
		for _, j := range allJobs {
			if rerunJobsIDSet.Contains(j.JobID) {
				continue
			}
			for _, need := range j.Needs {
				if rerunJobsIDSet.Contains(need) {
					found = true
					rerunJobs = append(rerunJobs, j)
					rerunJobsIDSet.Add(j.JobID)
					break
				}
			}
		}
		if !found {
			break
		}
	}

	return rerunJobs
}

// RerunJob resets a single completed job for rerun.
// If shouldBlock is true, the job will be set to StatusBlocked; otherwise StatusWaiting.
// Jobs that are not done are skipped (not an error).
// This only updates the job record; the caller is responsible for recomputing the
// run status and creating commit statuses after all jobs are updated.
func RerunJob(ctx context.Context, job *actions_model.ActionRunJob, shouldBlock bool) error {
	if !job.Status.IsDone() {
		return nil
	}

	status := job.Status
	initialStatus := actions_model.StatusWaiting
	if shouldBlock {
		initialStatus = actions_model.StatusBlocked
	}
	if err := job.PrepareNextAttempt(initialStatus); err != nil {
		return err
	}

	e := db.GetEngine(ctx)
	sess := e.ID(job.ID).
		Cols("attempt", "task_id", "status", "started", "stopped").
		Where(builder.Eq{"status": status})
	affected, err := sess.Update(job)
	if err != nil {
		return err
	}

	if affected != 0 && job.Status.IsWaiting() {
		if err := actions_model.IncreaseTaskVersion(ctx, job.OwnerID, job.RepoID); err != nil {
			return err
		}
	}

	return nil
}

// RerunRunJobs reruns jobs for a run.
// If jobID is 0, all jobs are rerun. Otherwise, only the specified job and its
// transitive dependents are rerun. If the run is done, its timing is reset.
// Returns the jobs that were rerun.
func RerunRunJobs(ctx context.Context, run *actions_model.ActionRun, jobID int64) ([]*actions_model.ActionRunJob, error) {
	// Reset run timing if the run is done
	if run.Status.IsDone() {
		run.PreviousDuration = run.Duration()
		run.Started = 0
		run.Stopped = 0
		if err := actions_model.UpdateRunWithoutNotification(ctx, run, "started", "stopped", "previous_duration"); err != nil {
			return nil, err
		}
	}

	jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		return nil, err
	}

	var rerunJobs []*actions_model.ActionRunJob

	if err := db.WithTx(ctx, func(ctx context.Context) error {
		if jobID == 0 {
			// Rerun all jobs
			rerunJobs = jobs
			for _, j := range jobs {
				shouldBlock := len(j.Needs) > 0
				if err := RerunJob(ctx, j, shouldBlock); err != nil {
					return err
				}
			}
		} else {
			// Find the specific job
			var targetJob *actions_model.ActionRunJob
			for _, j := range jobs {
				if j.ID == jobID {
					targetJob = j
					break
				}
			}
			if targetJob == nil {
				return nil
			}

			// Rerun the target job and its transitive dependents
			rerunJobs = GetAllRerunJobs(targetJob, jobs)
			for _, j := range rerunJobs {
				shouldBlock := j.JobID != targetJob.JobID
				if err := RerunJob(ctx, j, shouldBlock); err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	// Compute the final run status after all jobs have been updated
	updatedRun, columns, err := actions_model.ComputeRunStatus(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	if len(columns) > 0 {
		if err := UpdateRun(ctx, updatedRun, columns...); err != nil {
			return nil, err
		}
	}

	CreateCommitStatus(ctx, rerunJobs...)

	return rerunJobs, nil
}
