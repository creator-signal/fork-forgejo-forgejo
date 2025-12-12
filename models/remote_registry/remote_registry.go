// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remoteregistry

import (
	"context"
	"fmt"
	"net/url"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(RemoteRegistry))
}

// RemoteRegistryOwnerType represents the scope level of a remote registry configuration
type RemoteRegistryOwnerType string

// List of supported remote registry scopes
const (
	RRUser RemoteRegistryOwnerType = "user"
	RROrg  RemoteRegistryOwnerType = "org"
	RRRepo RemoteRegistryOwnerType = "repo"
)

func (rrt RemoteRegistryOwnerType) Name() string {
	switch rrt {
	case RRUser:
		return "user"
	case RROrg:
		return "org"
	case RRRepo:
		return "repo"
	}
	panic(fmt.Sprintf("unknown RemoteRegistryOwnerType: %s", string(rrt)))
}

// RemoteRegistry represents a remote OCI registry configuration
type RemoteRegistry struct {
	ID             int64                   `xorm:"pk autoincr"`
	Name           string                  `xorm:"UNIQUE(scope_name) NOT NULL"`
	OwnerType      RemoteRegistryOwnerType `xorm:"UNIQUE(scope_name) NOT NULL"`
	OwnerID        int64                   `xorm:"UNIQUE(scope_name) NOT NULL DEFAULT 0"`
	RemoteURL      string                  `xorm:"NOT NULL"`
	RemoteType     packages.Type           `xorm:"UNIQUE(s) INDEX NOT NULL"`
	RemoteUser     string                  `xorm:"TEXT"` // TODO: Is TEXT the right type for credentials?
	RemotePassword string                  `xorm:"TEXT"` // TODO: Password and Token encryption
	RemoteToken    string                  `xorm:"TEXT"` // TODO Setter and Getter for credentials
	CreatedUnix    timeutil.TimeStamp      `xorm:"created NOT NULL"`
	UpdatedUnix    timeutil.TimeStamp      `xorm:"updated NOT NULL"`
}

// TableName returns the table name for RemoteRegistry
func (RemoteRegistry) TableName() string {
	return "remote_registry"
}

type Credentials struct {
	RemoteUser     string
	RemotePassword string
	RemoteToken    string
}

func CreateRemoteRegistry(ctx context.Context, name, remoteURL string, remoteType packages.Type, cred *Credentials) {

	remoteRegistry = &RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
	}

}

func (rr RemoteRegistry) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(rr.Name, "Name")...)
	result = append(result, validation.ValidateNotEmpty(rr.OwnerType.Name(), "OwnerType")...)
	result = append(result, validation.ValidateNotEmpty(rr.OwnerID, "OwnderID")...)
	result = append(result, validation.ValidateNotEmpty(rr.RemoteType.Name(), "RemoteType")...)
	result = append(result, validation.ValidateNotEmpty(rr.CreatedUnix, "CreatedUnix")...)
	result = append(result, validation.ValidateNotEmpty(rr.UpdatedUnix, "UpdatedUnix")...)

	parsedURL, err := url.Parse(rr.RemoteURL)
	if err != nil {
		result = append(result, err.Error())
		return result
	}

	if parsedURL.Host == "" {
		result = append(result, "no host in Remote Registry URL given")
	}

	result = append(result, validation.ValidateOneOf(parsedURL.Scheme, []any{"http", "https"}, "parsedURL.Scheme")...)

	return result
}
