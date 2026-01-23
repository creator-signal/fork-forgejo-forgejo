package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/services/context"
)

func Dashboard(ctx *context.Context) {
	if !ctx.IsSigned {
		ctx.Redirect(ctx.Tr("sign_in_url"))
		return
	}

	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil {
		ctx.ServerError("GetUserRole", err)
		return
	}

	switch role {
	case edu.RoleTeacher, edu.RoleAdmin:
		ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/teacher/dashboard")
	case edu.RoleStudent:
		ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/student/assignments")
	default:
		ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/student/assignments")
	}
}
