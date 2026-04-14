package edu

import (
	"net/http"

	"forgejo.org/internal/edu"
	"forgejo.org/services/context"
)

// reqEduTeacher is middleware that requires the user to have a teacher or admin edu role.
// Forgejo site admins are always allowed (role hierarchy: site admin > edu admin > teacher > student).
func reqEduTeacher(ctx *context.Context) {
	if ctx.Doer.IsAdmin {
		return
	}
	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil || (role != edu.RoleTeacher && role != edu.RoleAdmin) {
		ctx.Error(http.StatusForbidden, "You must be a teacher or admin to access this page")
		return
	}
}

// reqEduAdmin is middleware that requires the user to be a Forgejo site admin or have edu admin role.
func reqEduAdmin(ctx *context.Context) {
	if ctx.Doer.IsAdmin {
		return
	}
	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil || role != edu.RoleAdmin {
		ctx.Error(http.StatusForbidden, "Admin access required")
		return
	}
}
