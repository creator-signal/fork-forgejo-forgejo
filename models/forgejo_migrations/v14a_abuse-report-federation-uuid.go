// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations

import (
	"database/sql"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add federation_uuid to abuse_report",
		Upgrade:     addUUIDAbuseReport,
	})
}

func addUUIDAbuseReport(x *xorm.Engine) error {
	type AbuseReport struct {
		FederationUUID sql.NullString `xorm:"DEFAULT NULL"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
	}, new(AbuseReport))

	return err
}
