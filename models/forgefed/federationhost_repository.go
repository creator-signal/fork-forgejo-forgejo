// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/models/federation_key"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
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

func findFederationHostFromDB(ctx context.Context, searchKey string, searchValue int64) (optional.Option[*FederationHost], error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where(searchKey, searchValue).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return optional.None[*FederationHost](), nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}

	return optional.Some(host), nil
}

func FindFederationHostByFqdnAndPort(ctx context.Context, fqdn string, port uint16) (optional.Option[*FederationHost], error) {
	host := new(FederationHost)
	has, err := db.GetEngine(ctx).Where("host_fqdn=? AND host_port=?", fqdn, port).Get(host)
	if err != nil {
		return nil, err
	} else if !has {
		return optional.None[*FederationHost](), nil
	}
	if res, err := validation.IsValid(host); !res {
		return nil, err
	}
	return optional.Some(host), nil
}

// FindFederationHostByPublicKey finds a [FederationHost] database entry by ActivityPub key ID.
//
// Returns:
//
// - (Option.Some(FederationHost), nil): success, a record was found
// - (Option.None, nil): success, no record found
// - (nil, error): failure, a database error occured
func FindFederationHostByKeyID(ctx context.Context, keyID federation_key.KeyID) (optional.Option[*FederationHost], error) {
	publicKey, err := federation_key.FindFederationPublicKey(ctx, keyID)
	if err != nil {
		return nil, err
	} else if !publicKey.Has() {
		return optional.None[*FederationHost](), nil
	}

	return findFederationHostFromDB(ctx, "id=?", publicKey.Value().ActorID)
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
