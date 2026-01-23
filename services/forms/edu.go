package forms

import "forgejo.org/modules/web"

// EduRoleForm form for admin role assignment
type EduRoleForm struct {
	Username string `binding:"Required"`
	Role     string `binding:"Required"`
}

func (f *EduRoleForm) Validate(ctx *web.ValidationContext) {
	// basic validation handled by binding
}
