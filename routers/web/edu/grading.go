package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplSubmissionDetail base.TplName = "edu/submission_detail"
)

func SubmissionDetail(ctx *context.Context) {
	ctx.Data["Title"] = "Submission Detail"
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	subID := ctx.ParamsInt64(":subID")

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
	ctx.Data["Assignment"] = assignment

	// Permission check
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
		ctx.Error(http.StatusForbidden, "Only instructors can view this page")
		return
	}

	submissions, err := svc.GetSubmissions(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetSubmissions", err)
		return
	}

	// Find the specific submission
	for _, s := range submissions {
		if s.ID == subID {
			ctx.Data["Submission"] = s

			// Load student info
			u, err := user_model.GetUserByID(ctx, s.UserID)
			if err != nil {
				log.Error("Failed to get user %d: %v", s.UserID, err)
			} else {
				ctx.Data["Student"] = u
			}

			// Load grader info
			if s.GradedByID > 0 {
				grader, err := user_model.GetUserByID(ctx, s.GradedByID)
				if err == nil {
					ctx.Data["Grader"] = grader
				}
			}

			// Load student repo link
			if s.StudentRepoID > 0 {
				studentRepo, err := repo_model.GetRepositoryByID(ctx, s.StudentRepoID)
				if err == nil && studentRepo != nil {
					ctx.Data["StudentRepoLink"] = studentRepo.FullName()
				}
			}

			break
		}
	}

	if ctx.Data["Submission"] == nil {
		ctx.NotFound("Submission not found", nil)
		return
	}

	testResults, err := svc.GetTestResults(ctx, subID)
	if err != nil {
		log.Error("Failed to get test results: %v", err)
	}
	ctx.Data["TestResults"] = testResults

	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplSubmissionDetail)
}

func GradeSubmissionPost(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	subID := ctx.ParamsInt64(":subID")

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

	// Permission check
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
		ctx.Error(http.StatusForbidden, "Only instructors can grade")
		return
	}

	grade := int(ctx.FormInt64("grade"))
	comment := ctx.FormString("comment")

	if grade < 0 || grade > 100 {
		ctx.Flash.Error("Grade must be between 0 and 100")
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions/" + ctx.Params(":subID"))
		return
	}

	if err := svc.GradeSubmission(ctx, subID, grade, comment, ctx.Doer.ID); err != nil {
		ctx.ServerError("GradeSubmission", err)
		return
	}

	ctx.Flash.Success("Grade saved successfully")
	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
}
