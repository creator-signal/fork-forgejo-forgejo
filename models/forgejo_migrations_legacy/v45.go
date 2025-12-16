// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations_legacy

import (
	"forgejo.org/models/packages"
	"forgejo.org/models/remote_registry"

	"xorm.io/xorm"
)

func AddRemoteRegistry(x *xorm.Engine) error {
	type RemoteRegistry struct {
		ID             int64                                   `xorm:"pk autoincr"`
		Name           string                                  `xorm:"UNIQUE NOT NULL"`
		OwnerType      remote_registry.RemoteRegistryOwnerType `xorm:"NOT NULL"`
		OwnerID        int64                                   `xorm:"NOT NULL"`
		RemoteURL      string                                  `xorm:"NOT NULL"`
		RemoteType     packages.Type                           `xorm:"INDEX NOT NULL"`
		RemoteUser     string                                  `xorm:"TEXT"` // TODO: Is TEXT the right type for credentials?
		RemotePassword string                                  `xorm:"TEXT"` // TODO: Password and Token encryption
		RemoteToken    string                                  `xorm:"TEXT"` // TODO Setter and Getter for credentials
	}

	return x.Sync(&RemoteRegistry{})
}
