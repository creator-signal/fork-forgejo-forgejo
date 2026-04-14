package edu

import (
	"forgejo.org/internal/edu"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

func Dashboard(ctx *context.Context) {
	if !ctx.IsSigned {
		ctx.Redirect(setting.AppSubURL + "/user/login")
		return
	}

	// Site admin always goes to admin panel
	if ctx.Doer.IsAdmin {
		ctx.Redirect(setting.AppSubURL + "/edu/admin")
		return
	}

	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetUserRole", err)
		return
	}

	switch role {
	case edu.RoleAdmin:
		ctx.Redirect(setting.AppSubURL + "/edu/admin")
	case edu.RoleTeacher:
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
	case edu.RoleTA:
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
	default:
		ctx.Redirect(setting.AppSubURL + "/edu/student/assignments")
	}
}

// setEduNavContext populates template data needed for the edu role-switch navbar.
// Call this in every edu handler before rendering HTML.
func setEduNavContext(ctx *context.Context) {
	role, _ := edu.GetUserRole(ctx, ctx.Doer.ID)
	ctx.Data["EduRole"] = role
	ctx.Data["IsSiteAdmin"] = ctx.Doer.IsAdmin
	ctx.Data["IsEduTA"] = role == edu.RoleTA
	ctx.Data["IsEduTeacher"] = role == edu.RoleTA || role == edu.RoleTeacher || role == edu.RoleAdmin || ctx.Doer.IsAdmin
	ctx.Data["IsFullTeacher"] = role == edu.RoleTeacher || role == edu.RoleAdmin || ctx.Doer.IsAdmin
	ctx.Data["IsEduAdmin"] = role == edu.RoleAdmin || ctx.Doer.IsAdmin
}
