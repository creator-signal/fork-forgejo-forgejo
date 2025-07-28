// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"database/sql"

	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add alternates tables",
		Upgrade:     addAlternatesTable,
	})
}

func addAlternatesTable(x *xorm.Engine) error {
	type Alternate struct {
		ID          int64              `xorm:"pk autoincr"`
		Name        string             `xorm:"NOT NULL UNIQUE"`
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}

	type Repository struct {
		AlternateID sql.NullInt64 `xorm:"INDEX DEFAULT NULL REFERENCES(alternate, id)"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Alternate), new(Repository))
	return err
}
