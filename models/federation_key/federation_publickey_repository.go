// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(FederationPublicKey))
}

// CreateFederationPublicKey creates a new `FederationPublicKey` database entry.
func CreateFederationPublicKey(ctx context.Context, key *FederationPublicKey) error {
	if key == nil {
		return fmt.Errorf("FederationPublicKey record is nil")
	} else if res, err := validation.IsValid(key); !res {
		return err
	}

	_, err := db.GetEngine(ctx).Insert(key)

	return err
}

// GetFederationPublicKey gets a `FederationPublicKey` entry by its database ID.
func GetFederationPublicKey(ctx context.Context, ID int64) (*FederationPublicKey, error) {
	key := new(FederationPublicKey)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(key)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationPublicKey record %v does not exist", ID)
	} else if res, err := validation.IsValid(key); !res {
		return nil, err
	}

	return key, nil
}

// FindFederationPublicKey gets a `FederationPublicKey` entry by its ActivityPub key ID.
//
// Returns:
//
// - (FederationPublicKey, nil): success, a record was found
// - (nil, nil): failure, no record found
// - (nil, error): failure, a database error occured
func FindFederationPublicKey(ctx context.Context, keyID string) (*FederationPublicKey, error) {
	key := new(FederationPublicKey)

	has, err := db.GetEngine(ctx).Where("key_id=?", keyID).Get(key)

	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	} else if res, err := validation.IsValid(key); !res {
		return nil, err
	}

	return key, nil
}

// FindOrCreateFederationPublicKey gets a `FederationPublicKey` entry by its ActivityPub key ID, or creates a new entry.
func FindOrCreateFederationPublicKey(ctx context.Context, key *FederationPublicKey) (*FederationPublicKey, error) {
	if key == nil {
		return nil, fmt.Errorf("FederationPublicKey is  null")
	}

	foundKey := new(FederationPublicKey)

	err := db.WithTx(ctx, func(ctx context.Context) error {
		eng := db.GetEngine(ctx)

		if has, err := eng.Where("key_id=?", key.KeyID).Get(foundKey); err != nil {
			return err
		} else if !has {
			if err = CreateFederationPublicKey(ctx, key); err != nil {
				return err
			} else if _, err = eng.Where("key_id=?", key.KeyID).Get(foundKey); err != nil {
				return err
			}
		} else if res, err := validation.IsValid(foundKey); !res {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return foundKey, nil
}
