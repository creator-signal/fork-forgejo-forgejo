// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/setting"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add AutoWatchOrgRepos to OrgUser",
		Upgrade:     addOrgUserAutoWatchOrgRepos,
	})
}

func addOrgUserAutoWatchOrgRepos(x *xorm.Engine) error {
	type OrgUser struct {
		ID                int64 `xorm:"pk autoincr"`
		AutoWatchOrgRepos bool  `xorm:"NOT NULL DEFAULT false"`
	}
	if _, err := x.SyncWithOptions(
		xorm.SyncOptions{IgnoreDropIndices: true},
		new(OrgUser),
	); err != nil {
		return err
	}

	if setting.Service.AutoWatchNewRepos {
		_, err := x.Exec("UPDATE `org_user` SET auto_watch_org_repos = ?", true)
		return err
	}
	return nil
}
