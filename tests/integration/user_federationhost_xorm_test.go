// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keyID = "https://forgejo.org/api/v1/activitypub/actor#meow"

func TestStoreFederationHost(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("ExplicitNull", func(t *testing.T) {
		federationHost := forgefed.FederationHost{
			HostFqdn: "ExplicitNull",
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federationHost)
		require.NoError(t, err)

		dbFederationHost := new(forgefed.FederationHost)
		has, err := db.GetEngine(db.DefaultContext).Where("host_fqdn=?", "ExplicitNull").Get(dbFederationHost)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("NotNull", func(t *testing.T) {
		key, err := util.RandomPKIXPublicKey(3072)
		require.NoError(t, err)

		federationHost := forgefed.FederationHost{
			HostFqdn: "ImplicitNull",
		}

		_, err = db.GetEngine(db.DefaultContext).Insert(&federationHost)
		require.NoError(t, err)

		dbFederationHost := new(forgefed.FederationHost)
		has, err := db.GetEngine(db.DefaultContext).Where("host_fqdn=?", "ImplicitNull").Get(dbFederationHost)
		require.NoError(t, err)
		assert.True(t, has)

		federationPublicKey, err := federation_key.NewFederationPublicKey(
			0,
			keyID,
			key,
			dbFederationHost.ID,
			federation_key.FederationHostType,
			federation_key.RsaSha256Cavage,
		)
		require.NoError(t, err)

		dbFederationPublicKey, err := federation_key.FindOrCreateFederationPublicKey(db.DefaultContext, federationPublicKey)
		require.NoError(t, err)

		assert.Equal(t, dbFederationPublicKey.ActorID, dbFederationHost.ID)

		assert.Nil(t, dbFederationPublicKey.Validate())
		assert.Equal(t, keyID, dbFederationPublicKey.KeyID.String())
		assert.Equal(t, key, dbFederationPublicKey.Key)
	})
}

func TestStoreFederatedUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("ExplicitNull", func(t *testing.T) {
		federatedUser := user.FederatedUser{
			UserID:           0,
			ExternalID:       "ExplicitNull",
			FederationHostID: 0,
		}

		_, err := db.GetEngine(db.DefaultContext).Insert(&federatedUser)
		require.NoError(t, err)

		dbFederatedUser := new(user.FederatedUser)
		has, err := db.GetEngine(db.DefaultContext).Where("user_id=?", 0).Get(dbFederatedUser)
		require.NoError(t, err)
		assert.True(t, has)
	})

	t.Run("NotNull", func(t *testing.T) {
		key, err := util.RandomPKIXPublicKey(3072)
		require.NoError(t, err)

		federatedUser := user.FederatedUser{
			UserID:           1,
			ExternalID:       "ImplicitNull",
			FederationHostID: 1,
		}

		_, err = db.GetEngine(db.DefaultContext).Insert(&federatedUser)
		require.NoError(t, err)

		dbFederatedUser := new(user.FederatedUser)
		has, err := db.GetEngine(db.DefaultContext).Where("user_id=?", 1).Get(dbFederatedUser)
		require.NoError(t, err)
		assert.True(t, has)

		federationPublicKey, err := federation_key.NewFederationPublicKey(
			0,
			keyID,
			key,
			dbFederatedUser.ID,
			federation_key.FederatedUserType,
			federation_key.RsaSha256Cavage,
		)
		require.NoError(t, err)

		dbFederationPublicKey, err := federation_key.FindOrCreateFederationPublicKey(db.DefaultContext, federationPublicKey)
		require.NoError(t, err)

		assert.Equal(t, dbFederationPublicKey.ActorID, dbFederatedUser.ID)

		assert.Nil(t, dbFederationPublicKey.Validate())
		assert.Equal(t, keyID, dbFederationPublicKey.KeyID.String())
		assert.Equal(t, key, dbFederationPublicKey.Key)
	})
}
