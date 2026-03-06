// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations

import (
	"fmt"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add social_fields and company_fields to user table",
		Upgrade:     addUserProfileFields,
	})
}

func addUserProfileFields(x *xorm.Engine) error {
	type User struct {
		SocialFields  string `xorm:"TEXT JSON"`
		CompanyFields string `xorm:"TEXT JSON"`
	}

	if err := x.Sync(new(User)); err != nil {
		return fmt.Errorf("Sync: %w", err)
	}
	return nil
}
