// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "create table avatar_vector, add field avatar_svg_hash to table user",
		Upgrade:     addAvatarVectorTable,
	})
}

func addAvatarVectorTable(x *xorm.Engine) error {
	type AvatarVector struct {
		ID      int64  `xorm:"pk autoincr"`
		UserID  int64  `xorm:"NOT NULL REFERENCES(user, id)"`
		SvgHash string `xorm:"VARBINARY(16)"`
		Svg     string `xorm:"TEXT"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(AvatarVector))
	return err
}
