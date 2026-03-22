package edu

import (
	"net/http"

	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

func BulkForkPost(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()
	if svc == nil {
		ctx.ServerError("getEduService", nil)
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
		ctx.Error(http.StatusForbidden, "Only instructors can perform bulk fork")
		return
	}

	task, err := svc.BulkForkForAssignment(ctx, assignmentID, ctx.Doer.ID)
	if err != nil {
		ctx.Flash.Error("Bulk fork failed: " + err.Error())
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
		return
	}

	if task.Failed > 0 {
		ctx.Flash.Warning("Bulk fork completed with errors: " + task.ErrorLog)
	} else {
		ctx.Flash.Success("Bulk fork completed successfully")
	}

	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
}

func BulkForkStatus(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()
	if svc == nil {
		ctx.ServerError("getEduService", nil)
		return
	}

	task, err := svc.GetBulkForkTaskByAssignment(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetBulkForkTaskByAssignment", err)
		return
	}

	if task == nil {
		ctx.JSON(http.StatusOK, map[string]any{
			"status":    "none",
			"total":     0,
			"completed": 0,
			"failed":    0,
		})
		return
	}

	ctx.JSON(http.StatusOK, map[string]any{
		"status":    task.Status,
		"total":     task.TotalUsers,
		"completed": task.Completed,
		"failed":    task.Failed,
	})
}
