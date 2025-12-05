// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

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
	db.RegisterModel(new(FederatedUser))
	db.RegisterModel(new(FederatedUserFollower))
}

func CreateFederatedUser(ctx context.Context, user *User, federatedUser *FederatedUser) error {
	if res, err := validation.IsValid(user); !res {
		return err
	}
	overwrite := CreateUserOverwriteOptions{
		IsActive:     optional.Some(false),
		IsRestricted: optional.Some(false),
	}

	// Begin transaction
	txCtx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err := committer.Close()
		if err != nil {
			log.Error("Error closing committer: %v", err)
		}
	}()

	if err := CreateUser(txCtx, user, &overwrite); err != nil {
		return err
	}

	federatedUser.UserID = user.ID
	if res, err := validation.IsValid(federatedUser); !res {
		return err
	}

	_, err = db.GetEngine(txCtx).Insert(federatedUser)
	if err != nil {
		return err
	}

	// Commit transaction
	return committer.Commit()
}

func FindFederatedUser(ctx context.Context, externalID string, federationHostID int64) (optional.Option[*User], optional.Option[*FederatedUser], error) {
	federatedUser := new(FederatedUser)
	user := new(User)

	var has bool

	eng := db.GetEngine(ctx)

	has, err := eng.Where("external_id=? and federation_host_id=?", externalID, federationHostID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return optional.None[*User](), optional.None[*FederatedUser](), nil
	}
	has, err = eng.ID(federatedUser.UserID).Get(user)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("FederatedUser table contains entry for user ID %v, but no user with this ID exists", federatedUser.UserID)
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, nil, err
	}
	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, nil, err
	}
	return optional.Some(user), optional.Some(federatedUser), nil
}

func GetFederatedUser(ctx context.Context, externalID string, federationHostID int64) (*User, *FederatedUser, error) {
	user, federatedUser, err := FindFederatedUser(ctx, externalID, federationHostID)
	if err != nil {
		return nil, nil, err
	} else if !federatedUser.Has() {
		return nil, nil, fmt.Errorf("FederatedUser not found (given externalId: %v, federationHostId: %v)", externalID, federationHostID)
	}
	return user.Value(), federatedUser.Value(), nil
}

func GetFederatedUserByUserID(ctx context.Context, userID int64) (*User, *FederatedUser, error) {
	federatedUser := new(FederatedUser)
	user := new(User)
	has, err := db.GetEngine(ctx).Where("user_id=?", userID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("FederatedUser table does not contain entry for user ID: %v", federatedUser.UserID)
	}
	has, err = db.GetEngine(ctx).ID(federatedUser.UserID).Get(user)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("FederatedUser table contains entry for user ID %v, but no user with this ID exists", federatedUser.UserID)
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, nil, err
	}
	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, nil, err
	}
	return user, federatedUser, nil
}

// FindFederatedUserByPublicKey finds a [FederatedUser] database entry by ActivityPub key ID.
//
// Returns:
//
// - (optional.Some(User), optional.Some(FederatedUser), nil): success, a record was found
// - (optional.None, optional.None, nil): success, no record found
// - (nil, nil, error): failure, a database error occured
func FindFederatedUserByKeyID(ctx context.Context, keyID federation_key.KeyID) (optional.Option[*User], optional.Option[*FederatedUser], error) {
	log.Trace("FindFederatedUserByKeyID: %v", keyID)
	federatedUser := new(FederatedUser)
	user := new(User)

	eng := db.GetEngine(ctx)
	publicKey, err := federation_key.FindFederationPublicKey(ctx, keyID)
	if err != nil {
		return nil, nil, err
	} else if !publicKey.Has() {
		return optional.None[*User](), optional.None[*FederatedUser](), nil
	}

	has, err := eng.Where("id=?", publicKey.Value().ActorID).Get(federatedUser)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return optional.None[*User](), optional.None[*FederatedUser](), nil
	}

	has, err = eng.ID(federatedUser.UserID).Get(user)
	if err != nil {
		return nil, nil, err
	} else if !has {
		return nil, nil, fmt.Errorf("FederatedUser table contains entry for user ID %v, but no user with this ID exists", federatedUser.UserID)
	} else if res, err := validation.IsValid(*user); !res {
		return nil, nil, err
	} else if res, err = validation.IsValid(*federatedUser); !res {
		return nil, nil, err
	}

	if res, err := validation.IsValid(*user); !res {
		return nil, nil, err
	}
	if res, err := validation.IsValid(*federatedUser); !res {
		return nil, nil, err
	}

	log.Trace("FindFederatedUserByKeyID: %v found user.ID %v, federated_user %v", keyID, user.ID, federatedUser)

	return optional.Some(user), optional.Some(federatedUser), nil
}

func DeleteFederatedUser(ctx context.Context, userID int64) error {
	_, err := db.GetEngine(ctx).Delete(&FederatedUser{UserID: userID})
	return err
}

func GetFollowersForUser(ctx context.Context, user *User) ([]*FederatedUserFollower, error) {
	if res, err := validation.IsValid(user); !res {
		return nil, err
	}
	followers := make([]*FederatedUserFollower, 0, 8)

	err := db.GetEngine(ctx).
		Where("followed_user_id = ?", user.ID).
		Find(&followers)
	if err != nil {
		return nil, err
	}
	for _, element := range followers {
		if res, err := validation.IsValid(*element); !res {
			return nil, err
		}
	}
	return followers, nil
}

func AddFollower(ctx context.Context, followedUser *User, followingUser *FederatedUser) (*FederatedUserFollower, error) {
	if res, err := validation.IsValid(followedUser); !res {
		return nil, err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return nil, err
	}

	federatedUserFollower, err := NewFederatedUserFollower(followedUser.ID, followingUser.UserID)
	if err != nil {
		return nil, err
	}
	_, err = db.GetEngine(ctx).Insert(&federatedUserFollower)
	if err != nil {
		return nil, err
	}

	return &federatedUserFollower, err
}

func RemoveFollower(ctx context.Context, followedUser *User, followingUser *FederatedUser) error {
	if res, err := validation.IsValid(followedUser); !res {
		return err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return err
	}

	_, err := db.GetEngine(ctx).Delete(&FederatedUserFollower{
		FollowedUserID:  followedUser.ID,
		FollowingUserID: followingUser.UserID,
	})
	return err
}

func IsFollowingAp(ctx context.Context, followedUser *User, followingUser *FederatedUser) (bool, error) {
	if res, err := validation.IsValid(followedUser); !res {
		return false, err
	}
	if res, err := validation.IsValid(followingUser); !res {
		return false, err
	}

	return db.GetEngine(ctx).Get(&FederatedUserFollower{
		FollowedUserID:  followedUser.ID,
		FollowingUserID: followingUser.UserID,
	})
}
