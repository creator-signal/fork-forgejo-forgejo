package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/services/context"
)

const (
	tplInstructorSubmissions base.TplName = "edu/instructor_submissions"
)

func InstructorSubmissions(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.instructor_panel")
	ctx.Data["PageIsEduAssignments"] = true

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

	if !perm.IsAdmin() && !perm.CanWrite(unit_model.TypeCode) && !isEduInstructor(ctx) {
		ctx.Error(http.StatusForbidden, "Only instructors can view this page")
		return
	}

	ctx.Data["Assignment"] = assignment

	submissions, err := svc.GetSubmissions(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetSubmissions", err)
		return
	}

	userIDs := make([]int64, 0, len(submissions))
	for _, s := range submissions {
		userIDs = append(userIDs, s.UserID)
	}
	users, err := user_model.GetUsersByIDs(ctx, userIDs)
	if err != nil {
		ctx.ServerError("GetUsersByIDs", err)
		return
	}
	userMap := make(map[int64]*user_model.User)
	for _, u := range users {
		userMap[u.ID] = u
	}

	ctx.Data["Submissions"] = submissions
	ctx.Data["UserMap"] = userMap

	// Build repo link map for student repos
	repoLinkMap := make(map[int64]string)
	for _, sub := range submissions {
		if sub.StudentRepoID > 0 {
			r, err := repo_model.GetRepositoryByID(ctx, sub.StudentRepoID)
			if err == nil && r != nil {
				repoLinkMap[sub.ID] = r.FullName()
			}
		}
	}
	ctx.Data["RepoLinkMap"] = repoLinkMap

	testResultMap := make(map[int64]*edu.TestResult)
	for _, sub := range submissions {
		tr, _ := svc.GetLatestTestResult(ctx, sub.ID)
		if tr != nil {
			testResultMap[sub.ID] = tr
		}
	}
	ctx.Data["TestResultMap"] = testResultMap

	bulkTask, _ := svc.GetBulkForkTaskByAssignment(ctx, assignmentID)
	ctx.Data["BulkForkTask"] = bulkTask

	syncTask, _ := svc.GetSyncForkTaskByAssignment(ctx, assignmentID)
	ctx.Data["SyncForkTask"] = syncTask

	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplInstructorSubmissions)
}
