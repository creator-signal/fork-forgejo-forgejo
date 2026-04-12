// Copyright 2014 The Gogs Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	"context"
	"strings"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"
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

type FindPublicKeySigningOptions struct {
	db.ListOptions
	OwnerID int64
}

func GetSignKeysForUser(ctx context.Context, user *user_model.User) ([]*PublicKeySigning, error) {
	return db.Find[PublicKeySigning](ctx, FindPublicKeySigningOptions{
		OwnerID: user.ID,
	})
}
