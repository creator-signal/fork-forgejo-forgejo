// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/models/gitea_migrations/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/timeutil"
	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "remove is_deleted column from activity action table",
		Upgrade:     removeIsDeletedColumnFromActivityActionTable,
	})
}

func removeIsDeletedColumnFromActivityActionTable(x *xorm.Engine) error {
	type Action struct {
		ID          int64 `xorm:"pk autoincr"`
		UserID      int64 `xorm:"INDEX"` // Receiver user id.
		ActUserID   int64 // Action user id.
		RepoID      int64
		CommentID   int64 `xorm:"INDEX"`
		IsDeleted   bool  `xorm:"NOT NULL DEFAULT false"`
		RefName     string
		IsPrivate   bool               `xorm:"NOT NULL DEFAULT false"`
		Content     string             `xorm:"TEXT"`
		CreatedUnix timeutil.TimeStamp `xorm:"created"`
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	activityActions := make([]*Action, 0)
	err := sess.OrderBy("id").Find(&activityActions)
	if err != nil {
		return err
	}

	for _, activityAction := range activityActions {
		if activityAction.IsDeleted {
			log.Debug("Permanently deleting Activity Action ID: %v", activityAction.ID)
			_, err := sess.Delete(activityAction)
			if err != nil {
				return err
			}
		}
	}

	if err := base.DropTableColumns(sess, "action", "is_deleted"); err != nil {
		return err
	}
	return sess.Commit()
}
