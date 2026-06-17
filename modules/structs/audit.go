// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package structs

import "time"

// AuditEvent represents a single security audit log event.
type AuditEvent struct {
	ID         int64  `json:"id"`
	Action     string `json:"action"`
	DoerID     int64  `json:"doer_id"`
	DoerName   string `json:"doer_name"`
	TargetType string `json:"target_type"`
	TargetName string `json:"target_name"`
	Message    string `json:"message"`
	IPAddress  string `json:"ip_address"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
}
