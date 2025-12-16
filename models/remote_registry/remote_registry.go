// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remote_registry

import (
	"context"
	"fmt"
	"net/url"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
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

var (
	ErrDuplicateRemoteRegistry = util.NewAlreadyExistErrorf("remote registry already exists")
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

func (rrt RemoteRegistryOwnerType) Valid() bool {
	if RemoteRegistryOwnerType(rrt.Name()) == RROrg ||
		RemoteRegistryOwnerType(rrt.Name()) == RRRepo ||
		RemoteRegistryOwnerType(rrt.Name()) == RRUser {
		return true
	}
	return false
}

// RemoteRegistry represents a remote OCI registry configuration
type RemoteRegistry struct {
	ID             int64                   `xorm:"pk autoincr"`
	Name           string                  `xorm:"UNIQUE NOT NULL"`
	OwnerType      RemoteRegistryOwnerType `xorm:"NOT NULL"`
	OwnerID        int64                   `xorm:"NOT NULL"`
	RemoteURL      string                  `xorm:"NOT NULL"`
	RemoteType     packages.Type           `xorm:"NOT NULL"`
	RemoteUser     string                  `xorm:"TEXT"` // TODO: Is TEXT the right type for credentials?
	RemotePassword string                  `xorm:"TEXT"` // TODO: Password and Token encryption
	RemoteToken    string                  `xorm:"TEXT"` // TODO Setter and Getter for credentials
}

// TableName returns the table name for RemoteRegistry
func (RemoteRegistry) TableName() string {
	return "remote_registry"
}

type RRCredentials struct {
	RemoteUser     string
	RemotePassword string
	RemoteToken    string
}

type RROpts struct {
	OwnerType RemoteRegistryOwnerType
	OwnerID   int64
	Auth      RRCredentials
}

func NewRemoteRegistry(name, remoteURL string, remoteType packages.Type, opts RROpts) (RemoteRegistry, error) {
	// decide whether repo, org, or user

	result := RemoteRegistry{
		Name:           name,
		RemoteURL:      remoteURL,
		RemoteType:     remoteType,
		OwnerType:      opts.OwnerType,
		OwnerID:        opts.OwnerID,
		RemoteUser:     opts.Auth.RemoteUser,
		RemotePassword: opts.Auth.RemotePassword,
		RemoteToken:    opts.Auth.RemoteToken,
	}

	if valid, err := validation.IsValid(result); !valid {
		return RemoteRegistry{}, err
	}
	return result, nil
}

// Create a remote registry in the DB, expects a valid rr
func CreateRemoteRegistry(ctx context.Context, rr RemoteRegistry) error {
	// Check if remote registry already exists
	existing := &RemoteRegistry{}
	exists, err := db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ? AND name = ?", rr.OwnerType, rr.OwnerID, rr.Name).
		Get(existing)
	if err != nil {
		return err
	}
	if exists {
		return ErrDuplicateRemoteRegistry
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	if _, err = db.GetEngine(ctx).Insert(rr); err != nil {
		return err
	}

	log.Info("Created remote registry %q (ID: %d) for owner_type %s:%d", rr.Name, rr.ID, rr.OwnerType, rr.OwnerID)
	return committer.Commit()
}

func (rr RemoteRegistry) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(rr.Name, "Name")...)
	result = append(result, validation.ValidateNotEmpty(rr.OwnerType.Name(), "OwnerType")...)
	result = append(result, validation.ValidateNotEmpty(rr.OwnerID, "OwnerID")...)
	result = append(result, validation.ValidateNotEmpty(rr.RemoteType.Name(), "RemoteType")...)

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

func GetRemoteRegistryByID(ctx context.Context, id int64) (*RemoteRegistry, error) {
	rr := &RemoteRegistry{}

	exists, err := db.GetEngine(ctx).ID(id).Get(rr)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRemoteRegistryNotExist
	}
	return rr, nil
}

// FindRemoteRegistryByName finds a remote registry by name within a scope
func FindRemoteRegistryByName(ctx context.Context, ownerType RemoteRegistryOwnerType, ownerID int64, name string) (*RemoteRegistry, error) {
	if !ownerType.Valid() {
		return nil, ErrInvalidRemoteRegistryOwner
	}

	rr := &RemoteRegistry{}
	exists, err := db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ? AND name = ?", ownerType, ownerID, name).
		Get(rr)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrRemoteRegistryNotExist
	}
	return rr, nil
}

// GetRemoteRegistriesByOwnerType gets all remote registries for a specific scope
func GetRemoteRegistriesByOwnerType(ctx context.Context, ownerType RemoteRegistryOwnerType, ownerID int64) ([]*RemoteRegistry, error) {
	if !ownerType.Valid() {
		return nil, ErrInvalidRemoteRegistryOwner
	}

	var remoteRegistries []*RemoteRegistry
	return remoteRegistries, db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		OrderBy("name ASC").
		Find(&remoteRegistries)
}
