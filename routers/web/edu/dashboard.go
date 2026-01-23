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

	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetUserRole", err)
		return
	}

	switch role {
	case edu.RoleTeacher, edu.RoleAdmin:
		ctx.Redirect(setting.AppSubURL + "/edu/teacher/dashboard")
	case edu.RoleStudent:
		ctx.Redirect(setting.AppSubURL + "/edu/student/assignments")
	default:
		ctx.Redirect(setting.AppSubURL + "/edu/student/assignments")
	}
}
