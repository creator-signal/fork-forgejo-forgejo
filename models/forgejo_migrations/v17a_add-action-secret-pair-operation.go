// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	secret_model "forgejo.org/models/secret"
	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add immutable repository Actions secret pair operations",
		Upgrade: addActionSecretPairOperation,
	})
}

func addActionSecretPairOperation(x *xorm.Engine) error {
	// Existing secret rows are not adopted; no original private intent is known.
	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(secret_model.ActionSecretPairOperation))
	return err
}
