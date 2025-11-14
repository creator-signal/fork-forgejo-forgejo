// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add fields avatar_svg and avatar_svg_hash to user table",
		Upgrade:     addAvatarSvgToUser,
	})
}

func addAvatarSvgToUser(x *xorm.Engine) error {
	type User struct {
		AvatarSVG     string `xorm:"TEXT"`
		AvatarSVGHash string `xorm:"VARCHAR(2048) NOT NULL"`
	}
	err := x.Sync(new(User))
	return err
}
