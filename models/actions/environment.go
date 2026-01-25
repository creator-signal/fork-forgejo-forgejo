// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

// ActionEnvironment represents a deployment environment for a repository.
// Environments group secrets and variables for deployment targets (e.g. production, staging).
//
// Currently only repository-level environments are implemented (OwnerID=0, RepoID=repo).
// The OwnerID field exists to support future org/user-level environments, following
// the same pattern as ActionVariable and Secret. When org-level environments are
// implemented, the convention will be:
//   - org/user level: OwnerID is org/user ID, RepoID is 0
//   - repo level: OwnerID is 0, RepoID is repo ID
//
// Both OwnerID and RepoID should never be non-zero simultaneously.
type ActionEnvironment struct {
	ID          int64              `xorm:"pk autoincr"`
	OwnerID     int64              `xorm:"UNIQUE(owner_repo_name)"`
	RepoID      int64              `xorm:"INDEX UNIQUE(owner_repo_name)"`
	Name        string             `xorm:"UNIQUE(owner_repo_name) NOT NULL"`
	Description string             `xorm:"TEXT"`
	CreatedUnix timeutil.TimeStamp `xorm:"created NOT NULL"`
	UpdatedUnix timeutil.TimeStamp `xorm:"updated"`
}

func init() {
	db.RegisterModel(new(ActionEnvironment))
}

// ErrEnvironmentNotFound represents an "environment not found" error.
type ErrEnvironmentNotFound struct {
	ID      int64
	OwnerID int64
	RepoID  int64
	Name    string
}

func (err ErrEnvironmentNotFound) Error() string {
	if err.ID > 0 {
		return fmt.Sprintf("environment not found [id: %d]", err.ID)
	}
	return fmt.Sprintf("environment not found [owner_id: %d, repo_id: %d, name: %s]", err.OwnerID, err.RepoID, err.Name)
}

func (err ErrEnvironmentNotFound) Unwrap() error {
	return util.ErrNotExist
}

// IsErrEnvironmentNotFound checks if an error is an ErrEnvironmentNotFound.
func IsErrEnvironmentNotFound(err error) bool {
	_, ok := err.(ErrEnvironmentNotFound)
	return ok
}

// FindEnvironmentOptions holds the options for finding environments
type FindEnvironmentOptions struct {
	db.ListOptions
	RepoID  int64
	OwnerID int64 // it will be ignored if RepoID is set
	Name    string
}

func (opts FindEnvironmentOptions) ToConds() builder.Cond {
	cond := builder.NewCond()

	cond = cond.And(builder.Eq{"repo_id": opts.RepoID})
	if opts.RepoID != 0 {
		// ignore OwnerID and treat it as 0
		cond = cond.And(builder.Eq{"owner_id": 0})
	} else {
		cond = cond.And(builder.Eq{"owner_id": opts.OwnerID})
	}

	if opts.Name != "" {
		cond = cond.And(builder.Eq{"name": strings.ToLower(opts.Name)})
	}

	return cond
}

var _ db.FindOptionsOrder = FindEnvironmentOptions{}

// ToOrders implements db.FindOptionsOrder
func (opts FindEnvironmentOptions) ToOrders() string {
	return "name, id"
}

// FindEnvironments returns a list of environments matching the given options
func FindEnvironments(ctx context.Context, opts FindEnvironmentOptions) ([]*ActionEnvironment, error) {
	return db.Find[ActionEnvironment](ctx, opts)
}

// CountEnvironments returns the count of environments matching the given options
func CountEnvironments(ctx context.Context, opts FindEnvironmentOptions) (int64, error) {
	return db.Count[ActionEnvironment](ctx, opts)
}

// GetEnvironmentByID returns an environment by its ID
func GetEnvironmentByID(ctx context.Context, id int64) (*ActionEnvironment, error) {
	env := &ActionEnvironment{}
	has, err := db.GetEngine(ctx).ID(id).Get(env)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrEnvironmentNotFound{ID: id}
	}
	return env, nil
}

// GetEnvironmentByName returns an environment by owner/repo and name.
// If repoID is set, ownerID is ignored and treated as 0.
func GetEnvironmentByName(ctx context.Context, ownerID, repoID int64, name string) (*ActionEnvironment, error) {
	if repoID != 0 {
		ownerID = 0
	}
	env := &ActionEnvironment{}
	has, err := db.GetEngine(ctx).Where("owner_id = ? AND repo_id = ? AND name = ?", ownerID, repoID, strings.ToLower(name)).Get(env)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrEnvironmentNotFound{OwnerID: ownerID, RepoID: repoID, Name: name}
	}
	return env, nil
}

// InsertEnvironment creates a new environment
func InsertEnvironment(ctx context.Context, ownerID, repoID int64, name, description string) (*ActionEnvironment, error) {
	if ownerID != 0 && repoID != 0 {
		// It's trying to create an environment that belongs to a repository, but OwnerID has been set accidentally.
		// Remove OwnerID to avoid confusion; it's not worth returning an error here.
		ownerID = 0
	}

	env := &ActionEnvironment{
		OwnerID:     ownerID,
		RepoID:      repoID,
		Name:        strings.ToLower(strings.TrimSpace(name)),
		Description: strings.TrimSpace(description),
	}
	return env, db.Insert(ctx, env)
}

// UpdateEnvironment updates an existing environment
func UpdateEnvironment(ctx context.Context, env *ActionEnvironment) (bool, error) {
	count, err := db.GetEngine(ctx).ID(env.ID).
		Where("owner_id = ? AND repo_id = ?", env.OwnerID, env.RepoID).
		Cols("name", "description").
		Update(&ActionEnvironment{
			Name:        strings.ToLower(strings.TrimSpace(env.Name)),
			Description: strings.TrimSpace(env.Description),
		})
	return count != 0, err
}

// DeleteEnvironment deletes an environment by ID and owner/repo
func DeleteEnvironment(ctx context.Context, id, ownerID, repoID int64) (bool, error) {
	count, err := db.GetEngine(ctx).
		Where("id = ? AND owner_id = ? AND repo_id = ?", id, ownerID, repoID).
		Delete(&ActionEnvironment{})
	return count != 0, err
}
