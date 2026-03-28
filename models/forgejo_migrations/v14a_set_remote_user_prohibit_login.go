// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"context"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"

	"xorm.io/builder"
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Set ProhibitLogin and UserTypeActivityPubUser for remote users created from ActivityPub.",
		Upgrade:     setProhibitLoginActivityPubUser,
	})
}

func setProhibitLoginActivityPubUser(x *xorm.Engine) error {
	return db.WithTx(db.DefaultContext, func(ctx context.Context) error {
		type User struct {
			ID int64 `xorm:"pk autoincr"`
		}
		return db.Iterate(ctx, builder.Eq{"type": 5}, func(ctx context.Context, user *User) error {
			log.Info("Checking if user %d is created from ActivityPub", user.ID)

			// Users created from f3 also have the RemoteUser user type. All
			// FederatedUser should reference exactly one User.
			has, err := db.GetEngine(ctx).Table("federated_user").Get(&user_model.FederatedUser{UserID: user.ID})
			if err != nil {
				return err
			}

			if !has {
				return nil
			}

			log.Info("Updating user %d", user.ID)
			_, err = db.GetEngine(ctx).Table("user").ID(user.ID).Cols("type", "prohibit_login", "passwd", "salt", "passwd_hash_algo").Update(&user_model.User{
				Type:           user_model.UserTypeActivityPubUser,
				ProhibitLogin:  true,
				Passwd:         "",
				Salt:           "",
				PasswdHashAlgo: "",
			})

			return err
		})
	})
}
