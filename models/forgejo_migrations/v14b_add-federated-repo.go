// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	ap "github.com/go-ap/activitypub"
	"xorm.io/xorm"
)

type FederatedRepository struct {
	ID        int64  `xorm:"pk autoincr"`
	OwnerID   int64  `xorm:"owner_id NOT NULL REFERENCES federated_user.id"`
	ObjectID  ap.ID  `xorm:"object_id NOT NULL"`
	Name      string `xorm:"name NOT NULL"`
	Summary   string `xorm:"summary NOT NULL"`
	Inbox     ap.IRI `xorm:"inbox"`
	Outbox    ap.IRI `xorm:"outbox"`
	Followers ap.IRI `xorm:"followers"`
	Team      ap.IRI `xorm:"team"`
}

func init() {
	registerMigration(&Migration{
		Description: "adds a separate federated_repository table",
		Upgrade:     v14AddFederatedRepositoryTable,
	})
}

func v14AddFederatedRepositoryTable(x *xorm.Engine) error {
	opts := xorm.SyncOptions{IgnoreDropIndices: true}

	_, err := x.SyncWithOptions(opts, new(FederationPublicKey))

	return err
}
