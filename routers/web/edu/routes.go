package edu

import (
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/services/context"
	"forgejo.org/services/forms"
)

func RegisterRoutes(m *web.Route, middlewares ...any) {
	m.Group("/edu", func() {
		m.Get("/dashboard", Dashboard)

		m.Group("/student", func() {
			m.Get("/assignments", Assignments)
			m.Get("/assignments/{id}", AssignmentDetail)
			m.Post("/assignments/{id}/join", JoinAssignment)
		})

		m.Group("/teacher", func() {
			m.Get("/assignments", Assignments)
			m.Get("/assignments/new", NewAssignment)
			m.Post("/assignments/new", NewAssignmentPost)
			m.Get("/assignments/{id}/submissions", InstructorSubmissions)
			m.Get("/dashboard", func(ctx *context.Context) {
				ctx.Redirect(setting.AppSubURL + "/edu/teacher/assignments")
			})
		})

		m.Group("/admin", func() {
			m.Get("", AdminPanel)
			m.Post("/roles", web.Bind(forms.EduRoleForm{}), UpdateUserRolePost)
		})
	}, middlewares...)
}
