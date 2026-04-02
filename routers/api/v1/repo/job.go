// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/actions"
	"forgejo.org/modules/util"
	"forgejo.org/services/context"
)

// GetJobLogs get the logs of the last job attempt
func GetJobLogs(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/jobs/{job_id}/logs repository JobLogs
	// ---
	// summary: Get the logs of the last job attempt
	// produces:
	// - text/plain
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: job_id
	//   in: path
	//   description: id of the job
	//   type: integer
	//   format: int64
	//   required: true
	// responses:
	//   "200":
	// 		 schema:
	// 		   type: file
	//   "400":
	//     "$ref": "#/responses/error"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	job, err := actions_model.GetRunJobByID(ctx, ctx.ParamsInt64(":job_id"))
	if err != nil {
		ctx.Error(http.StatusNotFound, "GetRunJobByID", err)
		return
	}

	if ctx.Repo.Repository.ID != job.RepoID {
		ctx.Error(http.StatusNotFound, "InvalidAccess", util.ErrNotExist)
		return
	}

	// Unlike job.Attempt, job.TaskID can be temporarily set to 0 in PrepareNextAttempt
	task, err := actions_model.GetTaskByJobAttempt(ctx, job.ID, job.Attempt)
	if err != nil || task.LogExpired {
		ctx.Error(http.StatusNotFound, "task could not be fetched", util.ErrNotExist)
		return
	}

	reader, err := actions.OpenLogs(ctx, task.LogInStorage, task.LogFilename)
	if err != nil {
		ctx.Error(http.StatusNotFound, "LogNotFound", util.ErrNotExist)
		return
	}
	defer reader.Close()

	ctx.ServeContent(reader, &context.ServeHeaderOptions{
		Filename:      fmt.Sprintf("%v.log", task.ID),
		ContentLength: &task.LogSize,
		ContentType:   "text/plain",
		Disposition:   "attachment",
	})
}
