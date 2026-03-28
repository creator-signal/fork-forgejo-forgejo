// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import "xorm.io/xorm"

func init() {
	registerMigration(&Migration{
		Description: "Add enable_file_age_color column to user table",
		Upgrade:     addSettingFileAgeColor,
	})
}

func addSettingFileAgeColor(x *xorm.Engine) error {
	type User struct {
		EnableFileAgeColor bool `xorm:"NOT NULL DEFAULT true"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(User))
	return err
}
