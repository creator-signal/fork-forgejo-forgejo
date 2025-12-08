// Copyright 2024,2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"testing"

	"forgejo.org/modules/util"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/require"
)

const keyID = "https://forgejo.org/api/v1/activitypub/actor#meow"

func Test_FederationPublicKeyValidation(t *testing.T) {
	ID := int64(1)
	actorID := int64(2)

	key, err := util.RandomPKIXPublicKey(3072)
	require.NoError(t, err)

	for _, actorType := range []ActorType{FederationHostType, FederatedUserType} {
		for _, alg := range []Algorithm{RsaSha256Cavage, RsaSha512Cavage, ED25519} {
			sut, err := NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				actorType,
				alg,
			)
			require.NoError(t, err, "sut should be valid")
			require.Equal(t, sut.KeyID.String(), keyID)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				Key:   key,
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			_, err = validation.IsValid(sut)
			require.NoError(t, err, "sut should be valid")
			require.Equal(t, sut.KeyID.String(), keyID)

			sut, err = NewFederationPublicKey(
				ID,
				"",
				key,
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", sut)

			sut = &FederationPublicKey{
				ID:  ID,
				Key: key,
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err := validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				keyID,
				[]byte{},
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				keyID,
				[]byte{0xb, 0xa, 0xd},
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				Key:   []byte{0xb, 0xa, 0xd},
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				"invalid,KeyID,URL",
				key,
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: "invalid,KeyID,URL",
				Key:   key,
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				keyID,
				key,
				0,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid actor ID, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				Key:   key,
				ActorID: 0,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid actor ID, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				ActorType("invalid_actor_type"),
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid actor type, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				Key:   key,
				ActorID: actorID,
				ActorType: ActorType("invalid_actor_type"),
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid actor type, but was valid %v", res)

			sut, err = NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				actorType,
				Algorithm("invalid-algorithm"),
			)
			require.Error(t, err, "sut should be invalid because of invalid signature algorithm, but was valid %v", res)

			sut = &FederationPublicKey{
				ID:    ID,
				KeyID: keyID,
				Key:   key,
				ActorID: actorID,
				ActorType: actorType,
				Algorithm: Algorithm("invalid-algorithm"),
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid signature algorithm, but was valid %v", res)
		}
	}
}
