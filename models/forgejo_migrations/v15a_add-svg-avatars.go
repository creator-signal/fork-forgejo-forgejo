// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "create table avatar_vector with fields svg and svg_hash, add field avatar_svg_hash to table user",
		Upgrade:     addAvatarVectorTable,
	})
}

func addAvatarVectorTable(x *xorm.Engine) error {
	type AvatarVector struct {
		SvgHash string `xorm:"VARBINARY(16)"`
		Svg     string `xorm:"TEXT"`
	}
	type User struct {
		AvatarSVGHash string `xorm:"VARBINARY(16)"`
	}
	err := x.Sync(new(AvatarVector), new(User))
	return err
}
