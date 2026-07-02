// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"fmt"

	"forgejo.org/modules/setting"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "widen external_login_user.provider to VARCHAR(255)",
		Upgrade:     expandExternalLoginUserProvider,
	})
}

// expandExternalLoginUserProvider widens external_login_user.provider from
// VARCHAR(25) to VARCHAR(255).
//
// The column stores the OAuth2 source *name* (AuthSourceProvider.Name(), the
// key goth is registered under), not the short provider type. A source whose
// name exceeds 25 characters therefore caused account linking to fail on
// backends that enforce column widths (PostgreSQL:
// `value too long for type character varying(25)`; MySQL truncates), silently
// dropping the external-login row.
func expandExternalLoginUserProvider(x *xorm.Engine) error {
	// SQLite does not enforce VARCHAR lengths, so there is nothing to alter.
	if setting.Database.Type.IsSQLite3() {
		return nil
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	var err error
	if setting.Database.Type.IsMySQL() {
		_, err = sess.Exec("ALTER TABLE `external_login_user` MODIFY COLUMN `provider` VARCHAR(255)")
	} else {
		_, err = sess.Exec("ALTER TABLE `external_login_user` ALTER COLUMN `provider` TYPE VARCHAR(255)")
	}
	if err != nil {
		return fmt.Errorf("failed to widen external_login_user.provider: %w", err)
	}

	return sess.Commit()
}
