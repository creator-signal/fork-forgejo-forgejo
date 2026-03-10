// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add first_dow column to user table",
		Upgrade:     addUserFirstDOW,
	})
}

func addUserFirstDOW(x *xorm.Engine) error {
	type User struct {
		FirstDOW int `xorm:"NOT NULL DEFAULT 1"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(User))
	return err
}
