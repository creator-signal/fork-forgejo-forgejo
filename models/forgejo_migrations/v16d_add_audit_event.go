// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add audit_event table",
		Upgrade:     addAuditEventTable,
	})
}

func addAuditEventTable(x *xorm.Engine) error {
	type AuditEvent struct {
		ID          int64  `xorm:"pk autoincr"`
		Action      string `xorm:"INDEX NOT NULL"`
		DoerID      int64  `xorm:"INDEX"`
		DoerName    string
		ScopeType   string
		ScopeID     int64
		TargetType  string
		TargetID    int64
		TargetName  string
		Message     string `xorm:"TEXT"`
		IPAddress   string
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(AuditEvent))
	return err
}
