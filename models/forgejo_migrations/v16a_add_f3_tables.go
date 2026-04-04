// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"time"

	"forgejo.org/modules/timeutil"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add F3 tables",
		Upgrade:     v16AddF3Tables,
	})
}

type v16Forge struct { //revive:disable-line:exported
	ID    int64  `xorm:"pk autoincr"`
	URL   string `xorm:"INDEX UNIQUE"`
	Type  string
	Token []byte `xorm:"BLOB"` // encrypted data
}

func (o *v16Forge) TableName() string {
	return "f3_forge"
}

type v16Mirror struct {
	ID                int64  `xorm:"pk autoincr"`
	Token             []byte `xorm:"BLOB"` // encrypted data
	ForgeID           int64  `xorm:"INDEX UNIQUE(f3_mirror_index) REFERENCES(f3_forge, id)"`
	FromPath          string `xorm:"INDEX UNIQUE(f3_mirror_index)"`
	ToPath            string
	Since             timeutil.TimeStamp
	Interval          time.Duration
	SendNotifications bool
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX"`
	NextUpdateUnix    timeutil.TimeStamp `xorm:"INDEX"`
	Err               string
	ErrMessage        string
}

func (o *v16Mirror) TableName() string {
	return "f3_mirror"
}

type v16Kind int8

type v16Resource struct {
	ID         int64   `xorm:"pk autoincr"`
	ForgeID    int64   `xorm:"INDEX UNIQUE(f3_resource_index) REFERENCES(f3_forge, id)"`
	ResourceID int64   `xorm:"INDEX UNIQUE(f3_resource_index)"`
	Kind       v16Kind `xorm:"INDEX UNIQUE(f3_resource_index)"`
}

func (o *v16Resource) TableName() string {
	return "f3_resource"
}

func v16AddF3Tables(x *xorm.Engine) error {
	if _, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v16Forge)); err != nil {
		return err
	}

	if _, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v16Mirror)); err != nil {
		return err
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v16Resource))
	return err
}
