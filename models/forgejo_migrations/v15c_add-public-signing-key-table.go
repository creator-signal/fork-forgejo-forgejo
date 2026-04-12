// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/timeutil"
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add a table for SSH signing keys",
		Upgrade:     up,
	})
}

func up(x *xorm.Engine) error {
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

	if err := x.Sync(new(PublicKeySigning)); err != nil {
		return err
	}

	// public user keys (type=1) can only be used for signing when the verified flag is true
	// copy all verified ones into the new table to make the transition for users smooth
	_, err := x.Exec(`
		INSERT INTO public_key_signing (
			owner_id,
			name,
			fingerprint,
			content,
			created_unix,
			updated_unix,
			verified
		)
		SELECT
			owner_id,
			name,
			fingerprint,
			content,
			created_unix,
			updated_unix,
			verified
		FROM public_key
		WHERE type = 1 AND verified;`)
	return err
}
