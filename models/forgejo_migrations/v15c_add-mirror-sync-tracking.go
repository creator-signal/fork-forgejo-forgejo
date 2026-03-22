// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add failed_sync_count and enabled fields to mirror and push_mirror tables",
		Upgrade:     addMirrorSyncTracking,
	})
}

func addMirrorSyncTracking(x *xorm.Engine) error {
	type Mirror struct {
		FailedSyncCount int  `xorm:"NOT NULL DEFAULT 0"`
		Enabled         bool `xorm:"NOT NULL DEFAULT true"`
	}

	type PushMirror struct {
		FailedSyncCount int  `xorm:"NOT NULL DEFAULT 0"`
		Enabled         bool `xorm:"NOT NULL DEFAULT true"`
	}

	if _, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Mirror)); err != nil {
		return err
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(PushMirror))
	return err
}
