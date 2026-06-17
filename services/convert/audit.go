// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	audit_model "forgejo.org/models/audit"
	api "forgejo.org/modules/structs"
)

// ToAuditEvent converts an audit log Event to its API representation.
func ToAuditEvent(e *audit_model.Event) *api.AuditEvent {
	return &api.AuditEvent{
		ID:         e.ID,
		Action:     string(e.Action),
		DoerID:     e.DoerID,
		DoerName:   e.DoerName,
		TargetType: e.TargetType,
		TargetName: e.TargetName,
		Message:    e.Message,
		IPAddress:  e.IPAddress,
		Created:    e.CreatedUnix.AsTime(),
	}
}
