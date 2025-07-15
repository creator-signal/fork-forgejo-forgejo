// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func AddResolvedUnixToAbuseReport(x *xorm.Engine) error {
	type AbuseReport struct {
		ResolvedUnix timeutil.TimeStamp
	}

	return x.Sync(&AbuseReport{})
}
