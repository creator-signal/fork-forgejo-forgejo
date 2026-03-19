// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add new columns (is_ca, principals) to public_key for user-defined SSH CAs",
		Upgrade:     addSSHCAFieldsToPublicKey,
	})
}

func addSSHCAFieldsToPublicKey(x *xorm.Engine) error {
	type PublicKey struct {
		IsCA       bool   `xorm:"NOT NULL DEFAULT false"`
		Principals string `xorm:"NOT NULL DEFAULT ''"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(PublicKey))
	return err
}
