package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

func AdminPanel(ctx *context.Context) {
	// Only Forgejo Admins
	if !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden)
		return
	}

	ctx.Data["Title"] = "Education Admin Panel"
	ctx.Data["PageIsEduAdmin"] = true

	// TODO: List users
	setEduNavContext(ctx)
	ctx.HTML(http.StatusOK, "edu/admin_panel")
}

func UpdateUserRolePost(ctx *context.Context) {
	// Only Forgejo Admins
	if !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden)
		return
	}

	form := web.GetForm(ctx).(*EduRoleForm)
	
	username := form.Username
	role := form.Role

	u, err := edu.GetUserByName(ctx, username)
	if err != nil {
		ctx.Flash.Error("User not found: " + err.Error())
		ctx.Redirect(setting.AppSubURL + "/edu/admin")
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
		ctx.Redirect(setting.AppSubURL + "/edu/admin")
		return
	}

	if err := edu.SetUserRole(ctx, u.ID, r); err != nil {
		ctx.ServerError("SetUserRole", err)
		return
	}

	ctx.Flash.Success("Role updated for " + username)
	ctx.Redirect(setting.AppSubURL + "/edu/admin")
}
