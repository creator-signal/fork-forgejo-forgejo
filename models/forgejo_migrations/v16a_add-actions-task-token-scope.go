// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add token_scope to table action_task",
		Upgrade:     addActionTaskTokenScope,
	})
}

func addActionTaskTokenScope(x *xorm.Engine) error {
	type AccessTokenScope string
	type ActionTask struct {
		TokenScope AccessTokenScope
	}

	_, err := x.SyncWithOptions(
		xorm.SyncOptions{
			IgnoreDropIndices: true,
			IgnoreConstrains:  true,
			IgnoreIndices:     true,
		},
		new(ActionTask),
	)
	return err
}
