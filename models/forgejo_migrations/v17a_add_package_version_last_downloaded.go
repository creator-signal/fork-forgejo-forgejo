// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add LastDownload to PackageVersion",
		Upgrade:     addLastDownloadToPackageVersion,
	})
}

func addLastDownloadToPackageVersion(x *xorm.Engine) error {
	type PackageVersion struct {
		ID           int64              `xorm:"pk autoincr"`
		LastDownload timeutil.TimeStamp `xorm:"INDEX NULL"`
	}

	_, err := x.SyncWithOptions(
		xorm.SyncOptions{IgnoreDropIndices: true},
		new(PackageVersion),
	)
	return err
}
