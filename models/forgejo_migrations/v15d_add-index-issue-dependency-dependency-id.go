// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add index on issue_dependency.dependency_id for blocks lookups",
		Upgrade:     addIssueDependencyDependencyIDIndex,
	})
}

func addIssueDependencyDependencyIDIndex(x *xorm.Engine) error {
	type IssueDependency struct {
		DependencyID int64 `xorm:"INDEX"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(IssueDependency))
	return err
}
