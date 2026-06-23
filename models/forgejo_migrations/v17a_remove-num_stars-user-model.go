// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"context"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "remove num_stars column from user table",
		Upgrade:     removeNumStarsFromUser,
	})
}

func removeNumStarsFromUser(x *xorm.Engine) error {
	exists, err := x.Dialect().IsColumnExist(context.Background(), x.DB(), "user", "num_stars")
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	_, err = x.Exec("ALTER TABLE `user` DROP COLUMN `num_stars`")
	return err
}
