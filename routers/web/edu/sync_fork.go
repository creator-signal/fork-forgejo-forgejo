package edu

import (
	"fmt"
	"net/http"

	"forgejo.org/internal/edu"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

func SyncAllForksPost(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	assignment, err := svc.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetAssignmentByID", err)
		return
	}
	if assignment == nil {
		ctx.NotFound("Assignment not found", nil)
		return
	}

	repo, err := repo_model.GetRepositoryByID(ctx, assignment.RepoID)
	if err != nil {
		ctx.ServerError("GetRepositoryByID", err)
		return
	}

	perm, err := access_model.GetUserRepoPermission(ctx, repo, ctx.Doer)
	if err != nil {
		ctx.ServerError("GetUserRepoPermission", err)
		return
	}
	if !perm.IsAdmin() && !perm.CanWrite(unit_model.TypeCode) {
		ctx.Error(http.StatusForbidden, "Only instructors can sync forks")
		return
	}

	task, err := svc.SyncAllForks(ctx, assignmentID, ctx.Doer.ID)
	if err != nil {
		ctx.Flash.Error("Sync forks failed: " + err.Error())
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
		return
	}

	ctx.Flash.Success(fmt.Sprintf("Sync complete: %d synced, %d skipped, %d failed", task.Synced, task.Skipped, task.Failed))
	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
}

func SyncForkStatus(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	task, err := svc.GetSyncForkTaskByAssignment(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetSyncForkTaskByAssignment", err)
		return
	}

	if task == nil {
		ctx.JSON(http.StatusOK, map[string]any{
			"status":  "none",
			"total":   0,
			"synced":  0,
			"skipped": 0,
			"failed":  0,
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"status":  task.Status,
		"total":   task.TotalRepos,
		"synced":  task.Synced,
		"skipped": task.Skipped,
		"failed":  task.Failed,
	})
}
