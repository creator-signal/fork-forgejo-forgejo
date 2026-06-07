// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "code.forgejo.org/xorm/xorm"

func init() {
	registerMigration(&Migration{
		Description: "add enable_shortcuts column to user",
		Upgrade:     AddEnableShortcutsToUser,
	})
}

func AddEnableShortcutsToUser(x *xorm.Engine) error {
	type User struct {
		ID              int64 `xorm:"pk autoincr"`
		EnableShortcuts bool  `xorm:"NOT NULL DEFAULT true"`
	}
	_, err := x.SyncWithOptions(
		xorm.SyncOptions{IgnoreDropIndices: true},
		new(User),
	)
	return err
}
