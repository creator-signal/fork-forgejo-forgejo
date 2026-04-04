// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgejo_migrations

import (
	"time"

	"forgejo.org/modules/timeutil"

	"code.forgejo.org/xorm/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "add F3 tables",
		Upgrade:     v17AddF3Tables,
	})
}

type v17Forge struct { //revive:disable-line:exported
	ID   int64  `xorm:"pk autoincr"`
	URL  string `xorm:"INDEX UNIQUE"`
	Type string
}

func (o *v17Forge) TableName() string {
	return "f3_forge"
}

type v17Mirror struct {
	ID                int64  `xorm:"pk autoincr"`
	ForgeID           int64  `xorm:"INDEX UNIQUE(f3_mirror_index) REFERENCES(f3_forge, id)"`
	RemotePath        string `xorm:"INDEX UNIQUE(f3_mirror_index)"`
	RemoteToken       []byte `xorm:"BLOB"` // encrypted data
	LocalPath         string
	LocalToken        []byte `xorm:"BLOB"` // encrypted data
	LocalUserID       int64
	Since             timeutil.TimeStamp
	Interval          time.Duration
	SendNotifications bool
	UpdatedUnix       timeutil.TimeStamp `xorm:"INDEX"`
	NextUpdateUnix    timeutil.TimeStamp `xorm:"INDEX"`
}

func (o *v17Mirror) TableName() string {
	return "f3_mirror"
}

type v17Kind int8

type v17Resource struct {
	ID         int64   `xorm:"pk autoincr"`
	MirrorID   int64   `xorm:"INDEX UNIQUE(f3_resource_index) REFERENCES(f3_mirror, id)"`
	ResourceID int64   `xorm:"INDEX UNIQUE(f3_resource_index)"`
	Kind       v17Kind `xorm:"INDEX UNIQUE(f3_resource_index)"`
}

func (o *v17Resource) TableName() string {
	return "f3_resource"
}

func v17AddF3Tables(x *xorm.Engine) error {
	if _, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v17Forge)); err != nil {
		return err
	}

	if _, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v17Mirror)); err != nil {
		return err
	}

	_, err := x.SyncWithOptions(xorm.SyncOptions{IgnoreDropIndices: true}, new(v17Resource))
	return err
}
