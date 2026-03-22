package edu

import (
	"net/http"
	"time"

	"forgejo.org/internal/edu"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplAssignmentList        base.TplName = "edu/assignment_list"
	tplStudentAssignmentList base.TplName = "edu/student_assignment_list"
	tplAssignmentDetail      base.TplName = "edu/assignment_detail"
	tplAssignmentEdit        base.TplName = "edu/assignment_edit"
)

func getEduService() edu.EducationalService {
	repo := edu.NewRepository()
	adapter := edu.NewForgejoAdapter()
	return edu.NewService(repo, adapter, adapter)
}

func StudentAssignments(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignments")
	ctx.Data["PageIsEduAssignments"] = true

	svc := getEduService()

	assignments, err := svc.GetAssignmentsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetAssignmentsForUser", err)
		return
	}

	ctx.Data["Assignments"] = assignments
	ctx.HTML(http.StatusOK, tplStudentAssignmentList)
}

func TeacherAssignments(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignments")
	ctx.Data["PageIsEduAssignments"] = true

	svc := getEduService()

	assignments, err := svc.GetAssignmentsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetAssignmentsForUser", err)
		return
	}

	// Build a map of courseID -> course name for display
	courseMap := make(map[int64]*edu.Course)
	for _, a := range assignments {
		if a.CourseID > 0 {
			if _, ok := courseMap[a.CourseID]; !ok {
				course, err := svc.GetCourseByID(ctx, a.CourseID)
				if err == nil && course != nil {
					courseMap[a.CourseID] = course
				}
			}
		}
	}

	ctx.Data["Assignments"] = assignments
	ctx.Data["CourseMap"] = courseMap
	ctx.HTML(http.StatusOK, tplAssignmentList)
}

func AssignmentDetail(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignment_detail")
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()

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
	ctx.Data["DeadlinePassed"] = assignment.DeadlineUnix > 0 && time.Now().Unix() > assignment.DeadlineUnix

	repo := edu.NewRepository()
	submission, err := repo.GetSubmission(ctx, assignment.ID, ctx.Doer.ID)

	if err != nil {
		log.Error("Failed to get submission: %v", err)
	}
	ctx.Data["Submission"] = submission

	if submission != nil {
		latestResult, _ := svc.GetLatestTestResult(ctx, submission.ID)
		ctx.Data["LatestTestResult"] = latestResult
	}

	ctx.HTML(http.StatusOK, tplAssignmentDetail)
}

func JoinAssignment(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()

	_, err := svc.JoinAssignment(ctx, ctx.Doer, assignmentID)
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(setting.AppSubURL + "/edu/student/assignments/" + ctx.Params(":id"))
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/student/assignments/" + ctx.Params(":id"))
}

func NewAssignment(ctx *context.Context) {
	ctx.Data["Title"] = "New Assignment"
	ctx.Data["PageIsEduAssignments"] = true
	ctx.Data["CourseID"] = ctx.FormInt64("course_id")
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

	var deadlineUnix int64
	if deadlineStr != "" {
		t, err := time.Parse("2006-01-02T15:04", deadlineStr)
		if err != nil {
			log.Warn("Failed to parse deadline: %v", err)
		} else {
			deadlineUnix = t.Unix()
		}
	}

	svc := getEduService()

	courseID := ctx.FormInt64("course_id")

	opts := edu.CreateAssignmentOptions{
		CourseID:     courseID,
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

func EditAssignment(ctx *context.Context) {
	ctx.Data["Title"] = "Edit Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()

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
	ctx.HTML(http.StatusOK, tplAssignmentEdit)
}

func EditAssignmentPost(ctx *context.Context) {
	ctx.Data["Title"] = "Edit Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()

	assignment, err := svc.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetAssignmentByID", err)
		return
	}
	if assignment == nil {
		ctx.NotFound("Assignment not found", nil)
		return
	}

	assignment.Title = ctx.FormString("title")
	assignment.Description = ctx.FormString("description")

	if assignment.Title == "" {
		ctx.Data["Assignment"] = assignment
		ctx.RenderWithErr("Title is required.", tplAssignmentEdit, nil)
		return
	}

	deadlineStr := ctx.FormString("deadline")
	if deadlineStr != "" {
		t, err := time.Parse("2006-01-02T15:04", deadlineStr)
		if err == nil {
			assignment.DeadlineUnix = t.Unix()
		}
	} else {
		assignment.DeadlineUnix = 0
	}

	if err := svc.UpdateAssignment(ctx, assignment); err != nil {
		ctx.ServerError("UpdateAssignment", err)
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments/" + ctx.Params(":id") + "/submissions")
}

func DeleteAssignmentPost(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := getEduService()

	if err := svc.DeleteAssignment(ctx, assignmentID); err != nil {
		ctx.ServerError("DeleteAssignment", err)
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
}
