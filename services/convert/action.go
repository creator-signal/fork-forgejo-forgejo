// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	"context"

	actions_model "forgejo.org/models/actions"
	access_model "forgejo.org/models/perm/access"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
)

// ToActionRunJob convert actions_model.ActionRunJob to api.ActionRunJob
// the job needs LoadAttributes called first (to populate Run for HTMLURL)
func ToActionRunJob(ctx context.Context, job *actions_model.ActionRunJob) *api.ActionRunJob {
	if job == nil {
		return nil
	}

	htmlURL, err := job.HTMLURL(ctx)
	if err != nil {
		log.Error("ActionRunJob[%d].HTMLURL: %v", job.ID, err)
	}

	return &api.ActionRunJob{
		ID:      job.ID,
		RunID:   job.RunID,
		RepoID:  job.RepoID,
		OwnerID: job.OwnerID,
		Name:    job.Name,
		Needs:   job.Needs,
		RunsOn:  job.RunsOn,
		TaskID:  job.TaskID,
		Status:  job.Status.String(),
		Started: job.Started.AsTime(),
		Stopped: job.Stopped.AsTime(),
		Created: job.Created.AsTime(),
		Updated: job.Updated.AsTime(),
		HTMLURL: htmlURL,
	}
}

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
