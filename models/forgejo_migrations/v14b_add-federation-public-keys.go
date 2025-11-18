// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"context"
	"database/sql"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/models/gitea_migrations/base"

	xb "xorm.io/builder"
	"xorm.io/xorm"
)

type FederationPublicKey struct {
	ID    int64  `xorm:"pk"`
	KeyID string `xorm:"UNIQUE NOT NULL"`
	Key   []byte `xorm:"BLOB NOT NULL"`
}

type ActorKey struct {
	ID        int64                  `xorm:"pk"`
	KeyID     sql.NullString         `xorm:"UNIQUE"`
	PublicKey sql.Null[sql.RawBytes] `xorm:"BLOB"`
}

type (
	FederationHost ActorKey
	FederatedUser  ActorKey
)

type Keys interface {
	GetID() int64
	GetKeyID() string
	GetPublicKey() []byte
}

func (f FederationHost) GetID() int64 {
	return f.ID
}

func (f FederationHost) GetKeyID() string {
	return f.KeyID.String
}

func (f FederationHost) GetPublicKey() []byte {
	return f.PublicKey.V
}

func (f FederatedUser) GetID() int64 {
	return f.ID
}

func (f FederatedUser) GetKeyID() string {
	return f.KeyID.String
}

func (f FederatedUser) GetPublicKey() []byte {
	return f.PublicKey.V
}

func init() {
	registerMigration(&Migration{
		Description: "adds a separate federation_public_key table",
		Upgrade:     v14SeparateFederationPublicKeyTable,
	})
}

func v14AddFederationPublicKeyTable(x *xorm.Engine) error {
	opts := xorm.SyncOptions{IgnoreDropIndices: true}

	_, err := x.SyncWithOptions(opts, new(FederationPublicKey))

	return err
}

func v14AddPublicKeyIDColumns(x *xorm.Engine) error {
	type FederationHost struct {
		PublicKeyID sql.NullInt64 `xorm:"INDEX UNIQUE"`
	}

	type FederatedUser struct {
		PublicKeyID sql.NullInt64 `xorm:"INDEX UNIQUE"`
	}

	opts := xorm.SyncOptions{IgnoreDropIndices: true}

	_, err := x.SyncWithOptions(opts, new(FederationHost))
	if err != nil {
		return err
	}

	_, err = x.SyncWithOptions(opts, new(FederatedUser))

	return err
}

func v14CopyExistingFederationPublicKeys[Bean Keys](x *xorm.Engine, table string) error {
	cond := xb.Neq{"key_id": "NULL"}.And(xb.Neq{"public_key": "NULL"})
	if err := db.Iterate[Bean](context.Background(), cond, func(_ context.Context, bean *Bean) error {
		if bean == nil {
			return fmt.Errorf("null bean")
		}
		key := *bean
		fk := FederationPublicKey{KeyID: key.GetKeyID(), Key: key.GetPublicKey()}
		var err error

		if _, err = x.Insert(&fk); err != nil {
			return err
		}
		var pkID int64
		if _, err = x.SQL(xb.Select("id").From(table).Where(xb.Eq{"key_id": fk.KeyID})).Get(&pkID); err != nil {
			return err
		}
		if _, err = x.Exec("UPDATE "+table+" SET public_key_id=?,key_id=NULL,public_key=NULL WHERE id=?", pkID, key.GetID()); err != nil {
			return err
		}

		return err
	}); err != nil {
		return err
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	if err := base.DropTableColumns(sess, table, "key_id"); err != nil {
		return err
	}

	if err := base.DropTableColumns(sess, table, "public_key"); err != nil {
		return err
	}

	return sess.Commit()
}

func v14SeparateFederationPublicKeyTable(x *xorm.Engine) error {
	err := v14AddFederationPublicKeyTable(x)
	if err != nil {
		return err
	}

	if err = v14AddPublicKeyIDColumns(x); err != nil {
		return err
	}

	if err = v14CopyExistingFederationPublicKeys[FederatedUser](x, "federated_user"); err != nil {
		return err
	}

	err = v14CopyExistingFederationPublicKeys[FederationHost](x, "federation_host")

	return err
}
