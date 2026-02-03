// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "create repo_license table",
		Upgrade:     createRepoLicenseTable,
	})
}

func createRepoLicenseTable(x *xorm.Engine) error {
	type RepoLicense struct {
		ID          int64 `xorm:"pk autoincr"`
		RepoID      int64 `xorm:"UNIQUE(s) NOT NULL REFERENCES(repository, id)"`
		CommitID    string
		License     string             `xorm:"VARCHAR(255) UNIQUE(s) NOT NULL"`
		Path        string             `xorm:"UNIQUE(s) NOT NULL"`
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX CREATED"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX UPDATED"`
	}

	return x.Sync(new(RepoLicense))
}
