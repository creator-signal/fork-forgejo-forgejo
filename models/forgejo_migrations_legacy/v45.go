// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations_legacy

import (
	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"

	"xorm.io/xorm"
)

func AddRemoteRegistry(x *xorm.Engine) error {
	type RemoteRegistry struct {
		ID             int64                            `xorm:"pk autoincr"`
		Name           string                           `xorm:"UNIQUE NOT NULL"`
		OwnerType      rr_model.RemoteRegistryOwnerType `xorm:"NOT NULL"`
		OwnerID        int64                            `xorm:"NOT NULL"`
		RemoteURL      string                           `xorm:"NOT NULL"`
		RemoteHost     string                           `xorm:"NOT NULL"`
		RemotePort     uint16                           `xorm:"NOT NULL"`
		RemoteType     packages.Type                    `xorm:"NOT NULL"`
		RemoteUser     string                           `xorm:"NOT NULL"`
		RemotePassword []byte                           `xorm:"BLOB"`
		RemoteToken    []byte                           `xorm:"BLOB"`
	}

	return x.Sync(&RemoteRegistry{})
}
