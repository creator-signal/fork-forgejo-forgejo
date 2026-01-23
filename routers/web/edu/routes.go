package edu

import (
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
)

	m.Group("/edu", func() {
		m.Get("/dashboard", Dashboard)

		// Student Routes
		m.Group("/student", func() {
			m.Get("/assignments", Assignments)
			m.Get("/assignments/{id}", AssignmentDetail)
			m.Post("/assignments/{id}/join", JoinAssignment)
		})

		// Teacher Routes
		m.Group("/teacher", func() {
			// Reuse Assignments (maybe filter later)
			m.Get("/assignments", Assignments)
			m.Get("/assignments/new", NewAssignment)
			m.Post("/assignments/new", NewAssignmentPost)
			m.Get("/assignments/{id}/submissions", InstructorSubmissions)
			// Redirect dashboard to assignments for now
			m.Get("/dashboard", func(ctx *context.Context) {
				ctx.Redirect(ctx.Tr("AppSubUrl") + "/edu/teacher/assignments")
			})
		})

		// Admin Routes
		m.Group("/admin", func() {
			m.Get("", AdminPanel)
			m.Post("/roles", web.Bind(forms.EduRoleForm{}), UpdateUserRolePost)
		})

		// Legacy/Direct Access (optional, or redirect)
		// Keeping these for compatibility if needed, or removing them to enforce role paths.
		// Removing for now to enforce new structure.
	}, middlewares...)
}
