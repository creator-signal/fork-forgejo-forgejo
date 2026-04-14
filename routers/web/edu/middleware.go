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
	if err != nil || (role != edu.RoleTA && role != edu.RoleTeacher && role != edu.RoleAdmin) {
		ctx.Error(http.StatusForbidden, "You must be a teacher or admin to access this page")
		return
	}
}

// isFullTeacher returns true if the user is a teacher, edu admin, or site admin (NOT TA).
func isFullTeacher(ctx *context.Context) bool {
	if ctx.Doer.IsAdmin {
		return true
	}
	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil {
		return false
	}
	return role == edu.RoleTeacher || role == edu.RoleAdmin
}

// isEduInstructor returns true if the user has TA, teacher, admin, or site admin role.
func isEduInstructor(ctx *context.Context) bool {
	if ctx.Doer.IsAdmin {
		return true
	}
	role, err := edu.GetUserRole(ctx, ctx.Doer.ID)
	if err != nil {
		return false
	}
	return role == edu.RoleTA || role == edu.RoleTeacher || role == edu.RoleAdmin
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
