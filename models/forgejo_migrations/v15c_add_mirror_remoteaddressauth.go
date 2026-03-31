// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add remote_address_auth to mirror",
		Upgrade:     addMirrorRemoteAddressAuth,
	})
}

func addMirrorRemoteAddressAuth(x *xorm.Engine) error {
	type Mirror struct {
		RemoteAddressAuth []byte `xorm:"BLOB NULL"`
	}
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(Mirror))
	return err
}
