// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"context"
	"strings"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"xorm.io/builder"
)

// PublicKeySigning represents an SSH public key used for git object signatures.
type PublicKeySigning struct {
	ID          int64  `xorm:"pk autoincr"`
	OwnerID     int64  `xorm:"INDEX NOT NULL"`
	Name        string `xorm:"NOT NULL"`
	Fingerprint string `xorm:"INDEX NOT NULL"`
	Content     string `xorm:"MEDIUMTEXT NOT NULL"`

	CreatedUnix timeutil.TimeStamp `xorm:"created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
	// true if the user proved via WebUI that they possess the private key
	Verified bool `xorm:"NOT NULL DEFAULT false"`
}

// OmitEmail returns content of public key without email address.
func (key *PublicKeySigning) OmitEmail() string {
	return strings.Join(strings.Split(key.Content, " ")[:2], " ")
}

func init() {
	db.RegisterModel(new(PublicKeySigning))
}

// checks if this fingerprint is aleady in use by another user
func checkSigningKeyFingerprint(ctx context.Context, fingerprint string) error {
	has, err := db.Exist[PublicKeySigning](ctx, builder.Eq{"fingerprint": fingerprint})
	if err != nil {
		return err
	} else if has {
		return ErrKeyAlreadyExist{0, fingerprint, ""}
	}
	return nil
}

// AddPublicSigningKey adds new public signing key to the database.
func AddPublicSigningKey(ctx context.Context, ownerID int64, name, content string) (*PublicKeySigning, error) {
	log.Trace(content)

	fingerprint, err := CalcFingerprint(content)
	if err != nil {
		return nil, err
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return nil, err
	}
	defer committer.Close()

	if err := checkSigningKeyFingerprint(ctx, fingerprint); err != nil {
		return nil, err
	}

	// Keys need unique names for each user
	has, err := db.GetEngine(ctx).
		Where("owner_id = ? AND name = ?", ownerID, name).
		Get(new(PublicKeySigning))
	if err != nil {
		return nil, err
	} else if has {
		return nil, ErrKeyNameAlreadyUsed{ownerID, name}
	}

	key := &PublicKeySigning{
		OwnerID:     ownerID,
		Name:        name,
		Fingerprint: fingerprint,
		Content:     content,
	}

	if err = db.Insert(ctx, key); err != nil {
		return nil, err
	}

	return key, committer.Commit()
}

// GetPublicSigningKeyByID returns public signing key by given ID.
func GetPublicSigningKeyByID(ctx context.Context, keyID int64) (*PublicKeySigning, error) {
	key := new(PublicKeySigning)
	has, err := db.GetEngine(ctx).
		ID(keyID).
		Get(key)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, ErrKeyNotExist{keyID}
	}
	return key, nil
}

type FindPublicKeySigningOptions struct {
	db.ListOptions
	OwnerID int64
}

func GetSignKeysForUser(ctx context.Context, user *user_model.User) ([]*PublicKeySigning, error) {
	return db.Find[PublicKeySigning](ctx, FindPublicKeySigningOptions{
		OwnerID: user.ID,
	})
}
