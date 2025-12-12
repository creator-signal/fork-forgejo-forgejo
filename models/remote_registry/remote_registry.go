// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remoteregistry

import (
	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/modules/timeutil"
)

func init() {
	db.RegisterModel(new(RemoteRegistry))
}

// RemoteRegistryOwnerType represents the scope level of a remote registry configuration
type RemoteRegistryOwnerType string

// List of supported remote registry scopes
const (
	User RemoteRegistryOwnerType = "user"
	Org  RemoteRegistryOwnerType = "org"
	Repo RemoteRegistryOwnerType = "repo"
)

// RemoteRegistry represents a remote OCI registry configuration
type RemoteRegistry struct {
	ID                   int64                   `xorm:"pk autoincr"`
	Name                 string                  `xorm:"UNIQUE(scope_name) NOT NULL"`
	URL                  string                  `xorm:"NOT NULL"`
	Type                 packages.Type           `xorm:"UNIQUE(s) INDEX NOT NULL"`
	OwnerType            RemoteRegistryOwnerType `xorm:"UNIQUE(scope_name) NOT NULL"`
	OwnerID              int64                   `xorm:"UNIQUE(scope_name) NOT NULL DEFAULT 0"`
	RemoteUser           string                  `xorm:"NOT NULL"`
	RemotePassword       string                  `xorm:"NOT NULL"`
	RemoteToken          string                  `xorm:"NOT NULL"`
	RemotePasswdHashAlgo string                  `xorm:"NOT NULL DEFAULT 'argon2'"`
	CreatedUnix          timeutil.TimeStamp      `xorm:"created NOT NULL"`
	UpdatedUnix          timeutil.TimeStamp      `xorm:"updated NOT NULL"`
}

// TableName returns the table name for RemoteRegistry
func (RemoteRegistry) TableName() string {
	return "remote_registry"
}
