// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/models/migrations/base"

	"xorm.io/xorm"
)

func RemoveIsDeletedColumnFromActivityActionTable(x *xorm.Engine) error {
	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}
	if err := base.DropTableColumns(sess, "action", "is_deleted"); err != nil {
		return err
	}
	return sess.Commit()
}
