package edu

import (
	"net/http"
	"time"

	"forgejo.org/internal/edu"
	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"
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

func StudentAssignments(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignments")
	ctx.Data["PageIsEduAssignments"] = true

	svc := edu.GetService()

	assignments, err := svc.GetAssignmentsForUser(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetAssignmentsForUser", err)
		return
	}

	ctx.Data["Assignments"] = assignments
	ctx.Data["PageIsEduStudent"] = true
	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplStudentAssignmentList)
}

func TeacherAssignments(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignments")
	ctx.Data["PageIsEduAssignments"] = true

	svc := edu.GetService()

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
	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplAssignmentList)
}

func AssignmentDetail(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("edu.assignment_detail")
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := edu.GetService()

	assignment, err := svc.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		ctx.ServerError("GetAssignmentByID", err)
		return
	}
	if assignment == nil {
		ctx.NotFound("Assignment not found", nil)
		return
	}

	// Verify enrollment for students
	if assignment.CourseID > 0 {
		enrollment, err := edu.NewRepository().GetEnrollment(ctx, assignment.CourseID, ctx.Doer.ID)
		if err != nil {
			ctx.ServerError("GetEnrollment", err)
			return
		}
		if enrollment == nil {
			isTeacher, _ := edu.IsTeacher(ctx, ctx.Doer.ID)
			if !isTeacher && !ctx.Doer.IsAdmin {
				ctx.NotFound("Not enrolled in this course", nil)
				return
			}
		}
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

		if submission.StudentRepoID > 0 {
			studentRepo, err := repo_model.GetRepositoryByID(ctx, submission.StudentRepoID)
			if err == nil && studentRepo != nil {
				ctx.Data["StudentRepoLink"] = studentRepo.FullName()
			}
		}
	}

	ctx.Data["PageIsEduStudent"] = true
	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplAssignmentDetail)
}

func JoinAssignment(ctx *context.Context) {
	assignmentID := ctx.ParamsInt64(":id")
	svc := edu.GetService()

	_, err := svc.JoinAssignment(ctx, ctx.Doer, assignmentID)
	if err != nil {
		ctx.Flash.Error(err.Error())
		ctx.Redirect(setting.AppSubURL + "/edu/student/assignments/" + ctx.Params(":id"))
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/student/assignments/" + ctx.Params(":id"))
}

// loadCoursesAndRepos populates template data with courses list and repos for the selected course.
// If a course with OrgID is selected, repos from that org are shown; otherwise user's own repos.
func loadCoursesAndRepos(ctx *context.Context, svc edu.EducationalService, selectedCourseID int64) {
	courses, err := svc.GetCoursesForUser(ctx, ctx.Doer.ID)
	if err != nil {
		log.Error("Failed to get courses: %v", err)
	}
	ctx.Data["Courses"] = courses
	ctx.Data["SelectedCourseID"] = selectedCourseID

	// Determine which repos to show based on selected course's org
	var repos repo_model.RepositoryList
	var courseHasOrg bool
	if selectedCourseID > 0 {
		for _, c := range courses {
			if c.ID == selectedCourseID && c.OrgID > 0 {
				courseHasOrg = true
				repos, err = org_model.GetOrgRepositories(ctx, c.OrgID)
				if err != nil {
					log.Error("Failed to get org repos: %v", err)
				}
				break
			}
		}
	}
	if !courseHasOrg {
		// Show user's own repos only when course is not linked to an org
		repos, _, err = repo_model.GetUserRepositories(ctx, &repo_model.SearchRepoOptions{
			ListOptions: db.ListOptions{ListAll: true},
			Actor:       ctx.Doer,
			Private:     true,
		})
		if err != nil {
			log.Error("Failed to get user repos: %v", err)
		}
	}
	ctx.Data["Repos"] = repos
}

func NewAssignment(ctx *context.Context) {
	ctx.Data["Title"] = "New Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	svc := edu.GetService()
	selectedCourseID := ctx.FormInt64("course_id")
	loadCoursesAndRepos(ctx, svc, selectedCourseID)

	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, "edu/assignment_new")
}

func NewAssignmentPost(ctx *context.Context) {
	ctx.Data["Title"] = "New Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	svc := edu.GetService()

	title := ctx.FormString("title")
	description := ctx.FormString("description")
	repoID := ctx.FormInt64("repo_id")
	deadlineStr := ctx.FormString("deadline")
	courseID := ctx.FormInt64("course_id")

	if courseID == 0 {
		loadCoursesAndRepos(ctx, svc, courseID)
		ctx.RenderWithErr("Course is required.", "edu/assignment_new", nil)
		return
	}

	if title == "" || repoID == 0 {
		loadCoursesAndRepos(ctx, svc, courseID)
		ctx.RenderWithErr("Title and Template Repository are required.", "edu/assignment_new", nil)
		return
	}

	// Verify repo exists
	_, err := repo_model.GetRepositoryByID(ctx, repoID)
	if err != nil {
		loadCoursesAndRepos(ctx, svc, courseID)
		ctx.RenderWithErr("Repository not found.", "edu/assignment_new", nil)
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

	opts := edu.CreateAssignmentOptions{
		CourseID:     courseID,
		RepoID:       repoID,
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
	svc := edu.GetService()

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
	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, tplAssignmentEdit)
}

func EditAssignmentPost(ctx *context.Context) {
	ctx.Data["Title"] = "Edit Assignment"
	ctx.Data["PageIsEduAssignments"] = true

	assignmentID := ctx.ParamsInt64(":id")
	svc := edu.GetService()

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
	svc := edu.GetService()

	if err := svc.DeleteAssignment(ctx, assignmentID); err != nil {
		ctx.ServerError("DeleteAssignment", err)
		return
	}

	ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
}
