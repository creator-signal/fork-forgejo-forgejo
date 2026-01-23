package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
)

func AdminPanel(ctx *context.Context) {
	// Only Forgejo Admins
	if !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden)
		return
	}

	ctx.Data["Title"] = "Education Admin Panel"

	// TODO: List users
	ctx.HTML(http.StatusOK, "edu/admin_panel")
}

func UpdateUserRolePost(ctx *context.Context) {
	// Only Forgejo Admins
	if !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden)
		return
	}

	form := web.GetForm(ctx).(*forms.EduRoleForm)
	
	username := ctx.FormString("username")
	role := ctx.FormString("role")

	u, err := edu.GetUserByName(ctx, username)
	if err != nil {
		ctx.Flash.Error("User not found: " + err.Error())
		ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/admin")
		return
	}

	var r edu.RoleType
	switch role {
	case "teacher":
		r = edu.RoleTeacher
	case "student":
		r = edu.RoleStudent
	default:
		ctx.Flash.Error("Invalid role")
		ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/admin")
		return
	}

	if err := edu.SetUserRole(ctx, u.ID, r); err != nil {
		ctx.ServerError("SetUserRole", err)
		return
	}

	ctx.Flash.Success("Role updated for " + username)
	ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/admin")
}
