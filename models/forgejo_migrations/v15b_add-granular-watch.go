// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add granular watch settings to repos.",
		Upgrade:     addGranularWatchColumns,
	})
}

func addGranularWatchColumns(x *xorm.Engine) error {
	type Watch struct {
		// Watch everything as it has been before when a user watches a repo.
		GranularWatchIssues       bool `xorm:"Bool DEFAULT 1"`
		GranularWatchPullRequests bool `xorm:"Bool DEFAULT 1"`
		GranularWatchReleases     bool `xorm:"Bool DEFAULT 1"`
	}

	return x.Sync(new(Watch))
}
