// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "create snippet table",
		Upgrade:     createSnippetTable,
	})
}

func createSnippetTable(x *xorm.Engine) error {
	type Snippet struct {
		ID          int64  `xorm:"pk autoincr"`
		OwnerID     int64  `xorm:"INDEX REFERENCES(user, id)"`
		UUID        string `xorm:"UNIQUE"`
		Name        string
		Description string `xorm:"TEXT"`
		Visibility  int8
		Language    string
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
	}

	return x.Sync(new(Snippet))
}
