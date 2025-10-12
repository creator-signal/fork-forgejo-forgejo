// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations

import (
	"database/sql"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add federated_abuse_report",
		Upgrade:     addForwardedAbuseReport,
	})
}

func addForwardedAbuseReport(x *xorm.Engine) error {
	type ForwardedAbuseReport struct {
		UUID             string         `xorm:"pk"`
		FederationHostID int64          `xorm:"NOT NULL"`
		ActorID          string         `xorm:"NOT NULL"`
		ActivityPubIDs   sql.NullString `xorm:"DEFAULT NULL"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{
		IgnoreDropIndices: true,
	}, new(ForwardedAbuseReport))

	return err
}
