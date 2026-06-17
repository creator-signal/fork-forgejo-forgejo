// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	audit_model "forgejo.org/models/audit"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const tplAudit base.TplName = "admin/audit"

// Audit shows the security audit log for the site administrator.
func Audit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.audit")
	ctx.Data["PageIsAdminAudit"] = true

	total := audit_model.CountEvents(ctx)
	page := max(ctx.FormInt("page"), 1)

	events, err := audit_model.Events(ctx, page, setting.UI.Admin.NoticePagingNum)
	if err != nil {
		ctx.ServerError("Events", err)
		return
	}
	ctx.Data["Events"] = events
	ctx.Data["Total"] = total
	ctx.Data["Page"] = context.NewPagination(int(total), setting.UI.Admin.NoticePagingNum, page, 5)

	ctx.HTML(http.StatusOK, tplAudit)
}
