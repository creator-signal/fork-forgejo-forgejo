// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add issue parent id",
		Upgrade:     addIssueParentId,
	})
}

func addIssueParentId(x *xorm.Engine) error {
	type Issue struct {
		ParentIssueID *int64 `xorm:"'parent_id' null index"`
	}

	return x.Sync(new(Issue))
}
