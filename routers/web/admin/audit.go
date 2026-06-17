// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	audit_model "forgejo.org/models/audit"
	"forgejo.org/models/db"
	"forgejo.org/modules/base"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
)

const tplAudit base.TplName = "admin/audit"

// Audit shows the security audit log for the site administrator.
func Audit(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.audit")
	ctx.Data["PageIsAdminAudit"] = true

	page := max(ctx.FormInt("page"), 1)

	events, total, err := audit_model.FindEvents(ctx, audit_model.FindEventsOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: setting.UI.Admin.NoticePagingNum},
	})
	if err != nil {
		ctx.ServerError("FindEvents", err)
		return
	}
	ctx.Data["Events"] = events
	ctx.Data["Total"] = total
	ctx.Data["Page"] = context.NewPagination(int(total), setting.UI.Admin.NoticePagingNum, page, 5)

	ctx.HTML(http.StatusOK, tplAudit)
}
