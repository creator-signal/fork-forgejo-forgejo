// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	federation_key_model "forgejo.org/models/federation_key"
	"forgejo.org/modules/validation"
)

func init() {
	db.RegisterModel(new(FederationHost))
}

func GetFederationHost(ctx context.Context, ID int64) (*FederationHost, error) {
	log.Trace("GetFederationHost: %v", ID)
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("id=?", ID).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, fmt.Errorf("FederationInfo record %v does not exist", ID)
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	log.Trace("GetFederationHost: %v, got host %v", ID, host)
	return host, nil
}

func findFederationHostFromDB(ctx context.Context, searchKey string, searchValue int64) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where(searchKey, searchValue).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}

	return host, nil
}

func FindFederationHostByFqdnAndPort(ctx context.Context, fqdn string, port uint16) (*FederationHost, error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("host_fqdn=? AND host_port=?", fqdn, port).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return nil, nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return host, nil
}

// FindFederationHostByPublicKey finds a [FederationHost] database entry by ActivityPub key ID.
//
// Returns:
//
// - (FederationHost, nil): success, a record was found
// - (nil, nil): failure, no record found
// - (nil, error): failure, a database error occured
func FindFederationHostByKeyID(ctx context.Context, rawKeyID string) (*FederationHost, error) {
	keyID, err := federation_key_model.NewKeyID(rawKeyID)
	if err != nil {
		return nil, err
	}

	publicKey, err := federation_key_model.FindFederationPublicKey(ctx, keyID.String())
	if err != nil {
		return nil, err
	} else if publicKey == nil {
		return nil, nil
	}

	return findFederationHostFromDB(ctx, "public_key_id=?", publicKey.ID)
}

func CreateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).Insert(host)
	return err
}

func UpdateFederationHost(ctx context.Context, host *FederationHost) error {
	if res, err := validation.IsValid(host); !res {
		return err
	}
	_, err := db.GetEngine(ctx).ID(host.ID).Update(host)
	return err
}
