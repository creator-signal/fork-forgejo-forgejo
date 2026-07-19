// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add last_gc_unix column to repository table",
		Upgrade:     addLastGCUnixToRepository,
	})
}

func addLastGCUnixToRepository(x *xorm.Engine) error {
	type Repository struct {
		LastGCUnix int64 `xorm:"DEFAULT 0"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Repository))
	return err
}
