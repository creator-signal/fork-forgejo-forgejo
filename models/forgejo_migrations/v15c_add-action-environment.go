// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add action_environment table and environment column to action_run_job",
		Upgrade:     addActionEnvironment,
	})
}

func addActionEnvironment(x *xorm.Engine) error {
	type ActionEnvironment struct {
		ID          int64              `xorm:"pk autoincr"`
		OwnerID     int64              `xorm:"UNIQUE(owner_repo_name)"`
		RepoID      int64              `xorm:"INDEX UNIQUE(owner_repo_name)"`
		Name        string             `xorm:"UNIQUE(owner_repo_name) NOT NULL"`
		Description string             `xorm:"TEXT"`
		CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL"`
		UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
	}

	type ActionRunJob struct {
		Environment string `xorm:"VARCHAR(255)"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(ActionEnvironment), new(ActionRunJob))
	return err
}
