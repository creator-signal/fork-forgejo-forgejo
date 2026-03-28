package edu

import (
	"fmt"
	"io"
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const (
	tplImportUpload  base.TplName = "edu/import_upload"
	tplImportPreview base.TplName = "edu/import_preview"
	tplImportResult  base.TplName = "edu/import_result"
)

func ImportUpload(ctx *context.Context) {
	ctx.Data["Title"] = "Import Users"
	ctx.Data["PageIsEduCourses"] = true

	courseID := ctx.ParamsInt64(":id")
	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	ctx.Data["Course"] = course
	ctx.HTML(http.StatusOK, tplImportUpload)
}

func ImportUploadPost(ctx *context.Context) {
	ctx.Data["Title"] = "Import Users"
	ctx.Data["PageIsEduCourses"] = true

	courseID := ctx.ParamsInt64(":id")
	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	ctx.Data["Course"] = course

	file, _, err := ctx.Req.FormFile("csv_file")
	if err != nil {
		ctx.RenderWithErr("Please select a CSV file to upload.", tplImportUpload, nil)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		ctx.ServerError("ReadAll", err)
		return
	}

	mapping := edu.CSVColumnMapping{
		FullNameCol: int(ctx.FormInt64("fullname_col")),
		EmailCol:    int(ctx.FormInt64("email_col")),
		GroupCol:    int(ctx.FormInt64("group_col")),
		HasHeader:   ctx.FormString("has_header") == "on",
	}

	draft, err := svc.UploadCSV(ctx, courseID, ctx.Doer.ID, data, mapping)
	if err != nil {
		ctx.RenderWithErr("Failed to parse CSV: "+err.Error(), tplImportUpload, nil)
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/edu/teacher/courses/%d/import/%d/preview", setting.AppSubURL, courseID, draft.ID))
}

func ImportPreview(ctx *context.Context) {
	ctx.Data["Title"] = "Import Preview"
	ctx.Data["PageIsEduCourses"] = true

	courseID := ctx.ParamsInt64(":id")
	draftID := ctx.ParamsInt64(":draftID")

	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	ctx.Data["Course"] = course

	draft, rows, err := svc.GetImportDraft(ctx, draftID)
	if err != nil {
		ctx.ServerError("GetImportDraft", err)
		return
	}
	if draft == nil {
		ctx.NotFound("Draft not found", nil)
		return
	}

	ctx.Data["Draft"] = draft
	ctx.Data["Rows"] = rows
	ctx.Data["RowCount"] = len(rows)
	ctx.HTML(http.StatusOK, tplImportPreview)
}

func ImportUpdateRow(ctx *context.Context) {
	courseID := ctx.ParamsInt64(":id")
	draftID := ctx.ParamsInt64(":draftID")

	rowID := ctx.FormInt64("row_id")
	username := ctx.FormString("username")
	email := ctx.FormString("email")

	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	if err := svc.UpdateDraftRow(ctx, rowID, username, email); err != nil {
		ctx.ServerError("UpdateDraftRow", err)
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/edu/teacher/courses/%d/import/%d/preview", setting.AppSubURL, courseID, draftID))
}

func ImportExecutePost(ctx *context.Context) {
	ctx.Data["Title"] = "Import Result"
	ctx.Data["PageIsEduCourses"] = true

	courseID := ctx.ParamsInt64(":id")
	draftID := ctx.ParamsInt64(":draftID")

	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	ctx.Data["Course"] = course

	roleStr := ctx.FormString("default_role")
	var defaultRole edu.RoleType
	switch roleStr {
	case "teacher":
		defaultRole = edu.RoleTeacher
	default:
		defaultRole = edu.RoleStudent
	}

	result, err := svc.ExecuteImport(ctx, draftID, ctx.Doer.ID, defaultRole)
	if err != nil {
		ctx.ServerError("ExecuteImport", err)
		return
	}

	ctx.Data["Result"] = result
	ctx.HTML(http.StatusOK, tplImportResult)
}

func ImportDeletePost(ctx *context.Context) {
	courseID := ctx.ParamsInt64(":id")
	draftID := ctx.ParamsInt64(":draftID")

	svc := edu.GetService()
	if svc == nil {
		ctx.ServerError("GetService", nil)
		return
	}

	course, err := svc.GetCourseByID(ctx, courseID)
	if err != nil {
		ctx.ServerError("GetCourseByID", err)
		return
	}
	if course == nil {
		ctx.NotFound("Course not found", nil)
		return
	}

	if course.CreatorID != ctx.Doer.ID {
		ctx.Error(http.StatusForbidden, "You can only import users into your own courses")
		return
	}

	if err := svc.DeleteImportDraft(ctx, draftID); err != nil {
		ctx.ServerError("DeleteImportDraft", err)
		return
	}

	ctx.Redirect(fmt.Sprintf("%s/edu/teacher/courses/%d", setting.AppSubURL, courseID))
}
