// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package swagger

import (
	api "forgejo.org/modules/structs"
)

// AuditEventList
// swagger:response AuditEventList
type swaggerResponseAuditEventList struct {
	// in:body
	Body []api.AuditEvent `json:"body"`
}
