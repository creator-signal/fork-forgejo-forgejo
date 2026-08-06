// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add new deploy key permissions to existing deploy keys",
		Upgrade:     addDeployKeyPermissions,
	})
}

func addDeployKeyPermissions(x *xorm.Engine) error {
	type DeployKey struct {
		CanWriteTags bool `xorm:"DEFAULT true"`
		CanWriteCode bool `xorm:"DEFAULT true"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(DeployKey))
	return err
}
