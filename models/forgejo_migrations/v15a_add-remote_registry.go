// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/models/packages"
	remote_registry_model "forgejo.org/models/remote_registry"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add remote_registry to table",
		Upgrade:     addRemoteRegistry,
	})
}

func addRemoteRegistry(x *xorm.Engine) error {
	type RemoteRegistry struct {
		ID             int64                                         `xorm:"pk autoincr"`
		Name           string                                        `xorm:"UNIQUE NOT NULL"`
		OwnerType      remote_registry_model.RemoteRegistryOwnerType `xorm:"NOT NULL"`
		OwnerID        int64                                         `xorm:"NOT NULL"`
		RemoteURL      string                                        `xorm:"NOT NULL"`
		RemoteHost     string                                        `xorm:"NOT NULL"`
		RemotePort     uint16                                        `xorm:"NOT NULL"`
		RemoteType     packages.Type                                 `xorm:"NOT NULL"`
		RemoteUser     string                                        `xorm:"NOT NULL"`
		RemotePassword []byte                                        `xorm:"BLOB"`
		RemoteToken    []byte                                        `xorm:"BLOB"`
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, &RemoteRegistry{})
	return err
}
