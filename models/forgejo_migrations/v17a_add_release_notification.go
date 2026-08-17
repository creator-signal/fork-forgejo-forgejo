// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add release to notification.",
		Upgrade:     addReleaseNotification,
	})
}

func addReleaseNotification(x *xorm.Engine) error {
	type Notification struct {
		ReleaseID int64 `xorm:"INDEX REFERENCES(release, id)"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Notification))
	return err
}
