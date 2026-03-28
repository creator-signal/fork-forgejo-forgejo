package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/services/context"
)

// reqEduTeacher is middleware that requires the user to have a teacher or admin edu role.
func reqEduTeacher(ctx *context.Context) {
	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil || (role != edu.RoleTeacher && role != edu.RoleAdmin) {
		ctx.Error(http.StatusForbidden, "You must be a teacher or admin to access this page")
		return
	}
}

// reqEduAdmin is middleware that requires the user to be a Forgejo site admin.
func reqEduAdmin(ctx *context.Context) {
	if !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden, "Admin access required")
		return
	}
}
