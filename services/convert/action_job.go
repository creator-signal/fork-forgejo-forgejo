// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package convert

import (
	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/actions"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/context"
)

// ToActionJobResponse converts an ActionRunJob to an API response with steps
func ToActionJobResponse(ctx *context.APIContext, job *actions_model.ActionRunJob, jobs []*actions_model.ActionRunJob) (*api.ActionJobResponse, error) {
	resp := &api.ActionJobResponse{
		ID:        job.ID,
		RunID:     job.RunID,
		Name:      job.Name,
		Status:    job.Status.String(),
		JobID:     job.JobID,
		Needs:     job.Needs,
		TaskID:    job.TaskID,
		TotalJobs: len(jobs),
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
			if err := task.LoadAttributes(ctx); err == nil {
				steps := actions.FullSteps(task)
				resp.Steps = make([]*api.ActionJobStep, 0, len(steps))
				for _, step := range steps {
					resp.Steps = append(resp.Steps, &api.ActionJobStep{
						Name:    step.Name,
						Status:  step.Status.String(),
						Started: step.Started.AsTime(),
						Stopped: step.Stopped.AsTime(),
					})
				}
			}
		}
	}

	// Add run information
	if job.Run != nil {
		resp.Run = &api.ActionRunSummary{
			ID:     job.Run.ID,
			Title:  job.Run.Title,
			Status: job.Run.Status.String(),
		}
	}

	return resp, nil
}
