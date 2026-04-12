package edu

import (
	"net/http"

	"code.forgejo.org/go-chi/binding"
	"forgejo.org/modules/web/middleware"
	"forgejo.org/services/context"
)

// EduRoleForm form for admin role assignment
type EduRoleForm struct {
	Username string `binding:"Required"`
	Role     string
}

func (f *EduRoleForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetValidateContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
