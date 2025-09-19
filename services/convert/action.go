// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	"context"

	actions_model "forgejo.org/models/actions"
	access_model "forgejo.org/models/perm/access"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
	service_context "forgejo.org/services/context"
)

// ToActionRun convert actions_model.User to api.ActionRun
// the run needs all attributes loaded
func ToActionRun(ctx context.Context, run *actions_model.ActionRun, doer *user_model.User) *api.ActionRun {
	if run == nil {
		return nil
	}

	permissionInRepo, _ := access_model.GetUserRepoPermission(ctx, run.Repo, doer)

	return &api.ActionRun{
		ID:                run.ID,
		Title:             run.Title,
		Repo:              ToRepo(ctx, run.Repo, permissionInRepo),
		WorkflowID:        run.WorkflowID,
		Index:             run.Index,
		TriggerUser:       ToUser(ctx, run.TriggerUser, doer),
		ScheduleID:        run.ScheduleID,
		PrettyRef:         run.PrettyRef(),
		IsRefDeleted:      run.IsRefDeleted,
		CommitSHA:         run.CommitSHA,
		IsForkPullRequest: run.IsForkPullRequest,
		NeedApproval:      run.NeedApproval,
		ApprovedBy:        run.ApprovedBy,
		Event:             run.Event.Event(),
		EventPayload:      run.EventPayload,
		TriggerEvent:      run.TriggerEvent,
		Status:            run.Status.String(),
		Started:           run.Started.AsTime(),
		Stopped:           run.Stopped.AsTime(),
		Created:           run.Created.AsTime(),
		Updated:           run.Updated.AsTime(),
		Duration:          run.Duration(),
		HTMLURL:           run.HTMLURL(),
	}
}

// ToActionRunJobResponse converts an ActionRunJob to an API response with steps
func ToActionRunJobResponse(ctx *service_context.APIContext, job *actions_model.ActionRunJob) (*api.ActionRunJobResponse, error) {
	resp := &api.ActionRunJobResponse{
		ID:     job.ID,
		RunID:  job.RunID,
		Name:   job.Name,
		Status: job.Status.String(),
		JobID:  job.JobID,
		Needs:  job.Needs,
		TaskID: job.TaskID,
	}

	// Convert timestamps
	if job.Started != 0 {
		resp.Started = job.Started.AsTime()
	}
	if job.Stopped != 0 {
		resp.Stopped = job.Stopped.AsTime()
	}

	// Add steps if the job has started
	if job.TaskID > 0 {
		task, err := actions_model.GetTaskByID(ctx, job.TaskID)
		if err == nil {
			task.Job = job
			if err := task.LoadSteps(ctx); err == nil {
				steps := actions.FullSteps(task)
				resp.Steps = make([]*api.ActionRunJobStep, 0, len(steps))
				for _, step := range steps {
					resp.Steps = append(resp.Steps, &api.ActionRunJobStep{
						Name:    step.Name,
						Status:  step.Status.String(),
						Started: step.Started.AsTime(),
						Stopped: step.Stopped.AsTime(),
					})
				}
			} else {
				log.Warn("Failed to load steps for task %d: %v", job.TaskID, err)
			}
		} else {
			log.Warn("Failed to get task %d for job %d: %v", job.TaskID, job.ID, err)
		}
	}

	return resp, nil
}

// ToActionRunJobs converts ActionRunJob models to API format
func ToActionRunJobs(jobs []*actions_model.ActionRunJob) []*api.ActionRunJobResponse {
	result := make([]*api.ActionRunJobResponse, len(jobs))
	for i, job := range jobs {
		result[i] = &api.ActionRunJobResponse{
			ID:      job.ID,
			RunID:   job.RunID,
			Name:    job.Name,
			Status:  job.Status.String(),
			Started: job.Started.AsTime(),
			Stopped: job.Stopped.AsTime(),
			JobID:   job.JobID,
			Needs:   job.Needs,
			TaskID:  job.TaskID,
		}
	}
	return result
}
