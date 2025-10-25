// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add avatar_svg field to user table",
		Upgrade:     addAvatarSvgToUser,
	})
}

func addAvatarSvgToUser(x *xorm.Engine) error {
	type User struct {
		AvatarSVG string `xorm:"TEXT"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(User))
	return err
}
