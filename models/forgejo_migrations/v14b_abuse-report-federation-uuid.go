// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations

import (
	"database/sql"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add foreign key constraint to abuse_report federation_uuid",
		Upgrade:     addConstraintAbuseReportUUID,
	})
}

func addConstraintAbuseReportUUID(x *xorm.Engine) error {
	type AbuseReport struct {
		FederationUUID sql.NullString `xorm:"INDEX DEFAULT NULL REFERENCES(forwarded_abuse_report, uuid)"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
	}, new(AbuseReport))

	return err
}
