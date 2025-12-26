// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add watch_events column to watch table for granular notifications",
		Upgrade:     addWatchEventsToWatch,
	})
}

func addWatchEventsToWatch(x *xorm.Engine) error {
	type Watch struct {
		WatchEvents int64 `xorm:"BIGINT NOT NULL DEFAULT 7"`
	}

	return x.Sync(new(Watch))
}
