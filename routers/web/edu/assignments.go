package edu

import (
	"net/http"
	"time"

	"forgejo.org/internal/edu"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"

	"xorm.io/xorm"
)

const (
	tplAssignmentList   base.TplName = "edu/assignment_list"
	tplAssignmentDetail base.TplName = "edu/assignment_detail"
)

func getSQLRunner(ctx *context.Context) edu.SQLRunner {
	e := db.GetEngine(ctx)
	if sess, ok := e.(*xorm.Session); ok {
		if sess.Tx() != nil {
			// *core.Tx embeds *sql.Tx
			return sess.Tx().Tx
		}
		// *core.DB embeds *sql.DB
		return sess.Engine().DB().DB
	}
	if eng, ok := e.(*xorm.Engine); ok {
		return eng.DB().DB
	}
	return nil
}

func getEduService(ctx *context.Context) edu.EducationalService {
	runner := getSQLRunner(ctx)
	if runner == nil {
		return nil
	}
	repo := edu.NewRepository(runner)
	adapter := edu.NewForgejoAdapter()
	return edu.NewService(repo, adapter)
}

func Assignments(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignments")
	ctx.Data["PageIsEduAssignments"] = true

	svc := getEduService(ctx)
	if svc == nil {
		ctx.ServerError("getEduService", nil)
		return
	}

	// TODO: filter by course/org user is enrolled in.
	assignments, err := svc.GetAssignments(ctx, 0)
	if err != nil {
		ctx.ServerError("GetAssignments", err)
		return
	}

	ctx.Data["Assignments"] = assignments
	ctx.HTML(http.StatusOK, tplAssignmentList)
}

func AssignmentDetail(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignment_detail")
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService(ctx)
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

	ctx.Data["Assignment"] = assignment

	repo := edu.NewRepository(getSQLRunner(ctx))
	submission, err := repo.GetSubmission(ctx, assignment.ID, ctx.Doer.ID)

	if err != nil {
		log.Error("Failed to get submission: %v", err)
		// Assume no submission indstead of failing with error
	}
	ctx.Data["Submission"] = submission

	ctx.HTML(http.StatusOK, tplAssignmentDetail)
}

func JoinAssignment(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService(ctx)
	if svc == nil {
		ctx.ServerError("getEduService", nil)
		return
	}

	_, err := svc.JoinAssignment(ctx, ctx.Doer, assignmentID)
	if err != nil {
		ctx.ServerError("JoinAssignment", err)
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/student/assignments/" + ctx.Params(":id"))
}

func NewAssignment(ctx *context.Context) {
	ctx.Data["Title"] = "New Assignment"
	ctx.Data["PageIsEduAssignments"] = true
	ctx.HTML(http.StatusOK, "edu/assignment_new")
}

func NewAssignmentPost(ctx *context.Context) {
	ctx.Data["Title"] = "New Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	title := ctx.FormString("title")
	description := ctx.FormString("description")
	templateRepoName := ctx.FormString("template_repo")
	deadlineStr := ctx.FormString("deadline")

	if title == "" || templateRepoName == "" {
		ctx.RenderWithErr("Title and Template Repository are required.", "edu/assignment_new", nil)
		return
	}

	repo, err := repo_model.GetRepositoryByName(ctx, ctx.Doer.ID, templateRepoName)
	if err != nil {
		if repo_model.IsErrRepoNotExist(err) {
			ctx.RenderWithErr("Repository not found in your account.", "edu/assignment_new", nil)
			return
		}
		ctx.ServerError("GetRepositoryByName", err)
		return
	}

	// datetimeformat: YYYY-MM-DDTHH:MM
	var deadlineUnix int64
	if deadlineStr != "" {
		t, err := time.Parse("2006-01-02T15:04", deadlineStr)
		if err != nil {
			log.Warn("Failed to parse deadline: %v", err)
		} else {
			deadlineUnix = t.Unix()
		}
	}

	svc := getEduService(ctx)
	if svc == nil {
		ctx.ServerError("getEduService", nil)
		return
	}

	opts := edu.CreateAssignmentOptions{
		RepoID:       repo.ID,
		Title:        title,
		Description:  description,
		DeadlineUnix: deadlineUnix,
	}

	_, err = svc.CreateAssignment(ctx, opts)
	if err != nil {
		ctx.ServerError("CreateAssignment", err)
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
}
