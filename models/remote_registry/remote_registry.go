// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package remote_registry

import (
	"context"
	"fmt"
	"net/url"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/modules/keying"
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
	RRUser   RemoteRegistryOwnerType = "user"
	RROrg    RemoteRegistryOwnerType = "org"
	passwCol string                  = "remote_password"
	tokenCol string                  = "remote_token"
)

var (
	ErrDuplicateRemoteRegistry    = util.NewAlreadyExistErrorf("remote registry already exists")
	ErrRemoteRegistryNotExist     = util.NewNotExistErrorf("remote registry does not exist")
	ErrInvalidRemoteRegistryOwner = util.NewInvalidArgumentErrorf("remote registry owner was invalid")
	ValidOwnerTypes               = []any{RRUser, RROrg}
)

func (rrt RemoteRegistryOwnerType) Name() string {
	switch rrt {
	case RRUser:
		return "user"
	case RROrg:
		return "org"
	}
	panic(fmt.Sprintf("unknown RemoteRegistryOwnerType: %s", string(rrt)))
}

func (rrt RemoteRegistryOwnerType) Valid() bool {
	if RemoteRegistryOwnerType(rrt.Name()) == RROrg ||
		RemoteRegistryOwnerType(rrt.Name()) == RRUser {
		return true
	}
	return false
}

// RemoteRegistry represents a remote OCI registry configuration
// https://xorm.io/docs/chapter-02/4.columns/
type RemoteRegistry struct {
	ID             int64                   `xorm:"pk autoincr"`
	Name           string                  `xorm:"UNIQUE(s) INDEX NOT NULL"`
	OwnerType      RemoteRegistryOwnerType `xorm:"NOT NULL"`
	OwnerID        int64                   `xorm:"UNIQUE(s) index NOT NULL"`
	RemoteURL      string                  `xorm:"NOT NULL"`
	RemoteHost     string                  `xorm:"NOT NULL"`
	RemotePort     uint16                  `xorm:"NOT NULL"`
	RemoteType     packages.Type           `xorm:"NOT NULL"`
	RemoteUser     string                  `xorm:"NOT NULL"`
	RemotePassword []byte                  `xorm:"BLOB"`
	RemoteToken    []byte                  `xorm:"BLOB"`
}

// TableName returns the table name for RemoteRegistry
func (RemoteRegistry) TableName() string {
	return "remote_registry"
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

	if err = db.Insert(ctx, rr); err != nil {
		return fmt.Errorf("create remote registry: %w", err)
	}

	log.Info("Created remote registry %q (ID: %d) for owner_type %s:%d", rr.Name, rr.ID, rr.OwnerType, rr.OwnerID)
	return committer.Commit()
}

// Update a remote registry in the DB, expects a valid rr
func UpdateRemoteRegistry(ctx context.Context, rr RemoteRegistry, oldName string) error {
	// Check if remote registry already exists
	existing := &RemoteRegistry{}
	exists, err := db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ? AND name = ?", rr.OwnerType, rr.OwnerID, oldName).
		Get(existing)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRemoteRegistryNotExist
	}

	affected, err := db.GetEngine(ctx).ID(existing.ID).Update(rr)
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrRemoteRegistryNotExist
	}

	log.Info("Updated remote registry %q (ID: %d) for owner_type %s:%d", rr.Name, rr.ID, rr.OwnerType, rr.OwnerID)
	return nil
}

// Delete a remote registry in the DB, expects a valid rr
func DeleteRemoteRegistry(ctx context.Context, ownerType RemoteRegistryOwnerType, ownerID int64, registryName string) error {
	// Check if remote registry already exists
	existing := &RemoteRegistry{}
	exists, err := db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ? AND name = ?", ownerType, ownerID, registryName).
		Get(existing)
	if err != nil {
		return err
	}
	if !exists {
		return ErrRemoteRegistryNotExist
	}

	affected, err := db.GetEngine(ctx).ID(existing.ID).Delete(existing)
	if err != nil {
		return err
	}
	if affected != 1 {
		return ErrRemoteRegistryNotExist
	}

	log.Info("Deleted remote registry %q for owner_type %s:%d", registryName, ownerType, ownerID)
	return nil
}

func (rr *RemoteRegistry) SetRemotePassword(password string) {
	if password != "" {
		rr.RemotePassword = keying.RemoteRegistry.Encrypt([]byte(password), keying.ColumnAndID(passwCol, rr.OwnerID))
	} else {
		rr.RemotePassword = []byte{}
	}
}

func (rr *RemoteRegistry) SetRemoteToken(token string) {
	if token != "" {
		rr.RemoteToken = keying.RemoteRegistry.Encrypt([]byte(token), keying.ColumnAndID(tokenCol, rr.OwnerID))
	} else {
		rr.RemoteToken = []byte{}
	}
}

func (rr *RemoteRegistry) GetRemotePassword() (string, error) {
	key := keying.RemoteRegistry
	if len(rr.RemotePassword) > 0 {
		password, err := key.Decrypt(rr.RemotePassword, keying.ColumnAndID(passwCol, rr.OwnerID))
		if err != nil {
			log.Error("unable to decrypt remote password[id=%d,name=%q]: %v", rr.ID, rr.Name, err)
			return "", err
		}
		return string(password), nil
	}
	return "", nil
}

func (rr *RemoteRegistry) GetRemoteToken() (string, error) {
	key := keying.RemoteRegistry
	if len(rr.RemoteToken) > 0 {
		token, err := key.Decrypt(rr.RemoteToken, keying.ColumnAndID(tokenCol, rr.OwnerID))
		if err != nil {
			log.Error("unable to decrypt remote password[id=%d,name=%q]: %v", rr.ID, rr.Name, err)
			return "", err
		}
		return string(token), nil
	}
	return "", nil
}

func (rr RemoteRegistry) Validate() []string {
	var result []string
	result = append(result, validation.ValidateURLSafe(rr.Name)...)
	result = append(result, validation.ValidateNotEmpty(rr.OwnerID, "OwnerID")...)
	result = append(result, validation.ValidateNotEmpty(rr.RemoteType.Name(), "RemoteType")...)
	result = append(result, validation.ValidateOneOf(rr.OwnerType, ValidOwnerTypes, "OwnerType")...)

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

// GetRemoteRegistryByName finds a remote registry by name
func GetRemoteRegistryByName(ctx context.Context, ownerType RemoteRegistryOwnerType, ownerID int64, name string) (*RemoteRegistry, error) {
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

// GetRemoteRegistriesByOwnerType gets all remote registries
func GetRemoteRegistriesByOwnerType(ctx context.Context, ownerType RemoteRegistryOwnerType, ownerID int64) ([]*RemoteRegistry, error) {
	if !ownerType.Valid() {
		return nil, ErrInvalidRemoteRegistryOwner
	}

	var remoteRegistries []*RemoteRegistry
	err := db.GetEngine(ctx).
		Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).
		OrderBy("name ASC").
		Find(&remoteRegistries)
	if err != nil {
		return nil, err
	}

	return remoteRegistries, nil
}
