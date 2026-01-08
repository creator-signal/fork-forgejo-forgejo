// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(FederatedRepository))
}

// CreateFederatedRepository creates a new [FederatedRepository] database entry.
func CreateFederatedRepository(ctx context.Context, repo *FederatedRepository) error {
	if res, err := validation.IsValid(repo); !res {
		return err
	}
	return db.WithTx(ctx, func(txCtx context.Context) error {
		_, err := db.GetEngine(txCtx).Insert(repo)
		return err
	})
}

// GetFederatedRepository fetches a [FederatedRepository] record from the database.
func GetFederatedRepository(ctx context.Context, ID int64) (*FederatedRepository, error) {
	log.Trace("GetFederatedRepository: %v", ID)
	repo := new(FederatedRepository)

	if err := db.WithTx(ctx, func(txCtx context.Context) error {
		has, err := db.GetEngine(ctx).Where("id=?", ID).Get(repo)
		if err != nil {
			return err
		} else if !has {
			return fmt.Errorf("FederatedRepository record %v does not exist", ID)
		}

		if res, err := validation.IsValid(repo); !res {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	log.Trace("GetFederatedRepository: %v, got repo %v", ID, repo)
	return repo, nil
}

// FindFederatedRepository attempts to fetch a [FederatedRepository] record from the database.
//
// Returns [ErrFederatedRepoNotExist] if no record was found.
func FindFederatedRepository(ctx context.Context, name string, ownerID int64) (*FederatedRepository, error) {
	log.Trace("FindFederatedRepository, name: %s, owner ID: %d", name, ownerID)
	repo := new(FederatedRepository)

	if err := db.WithTx(ctx, func(txCtx context.Context) error {
		has, err := db.GetEngine(ctx).Where("name=? AND owner_id=?", name, ownerID).Get(repo)
		if err != nil {
			return err
		} else if !has {
			return ErrFederatedRepoNotExist{Name: name}
		}

		if res, err := validation.IsValid(repo); !res {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	log.Trace("FindFederatedRepository: %d, got repo %v", ownerID, repo)
	return repo, nil
}
