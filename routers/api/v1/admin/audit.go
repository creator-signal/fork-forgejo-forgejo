// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package admin

import (
	"net/http"

	audit_model "forgejo.org/models/audit"
	api "forgejo.org/modules/structs"
	"forgejo.org/routers/api/v1/utils"
	"forgejo.org/services/context"
	"forgejo.org/services/convert"
)

// ListAuditEvents lists security audit log events.
func ListAuditEvents(ctx *context.APIContext) {
	// swagger:operation GET /admin/audit admin adminListAuditEvents
	// ---
	// summary: List security audit log events
	// produces:
	// - application/json
	// parameters:
	// - name: action
	//   in: query
	//   description: only return events with this action identifier (e.g. user_login)
	//   type: string
	// - name: doer
	//   in: query
	//   description: only return events performed by this user ID
	//   type: integer
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/AuditEventList"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	events, total, err := audit_model.FindEvents(ctx, audit_model.FindEventsOptions{
		ListOptions: utils.GetListOptions(ctx),
		Action:      audit_model.Action(ctx.FormString("action")),
		DoerID:      ctx.FormInt64("doer"),
	})
	if err != nil {
		ctx.Error(http.StatusInternalServerError, "FindEvents", err)
		return
	}

	results := make([]*api.AuditEvent, len(events))
	for i := range events {
		results[i] = convert.ToAuditEvent(events[i])
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, &results)
}
