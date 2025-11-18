// Copyright 2024,2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key_test

import (
	"testing"

	"forgejo.org/models/federation_key"
	"forgejo.org/modules/util"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/require"
)

const keyID = "https://forgejo.org/api/v1/activitypub/actor#meow"

func Test_FederationPublicKeyValidation(t *testing.T) {
	ID := int64(1)
	actorID := int64(2)

	for _, actorType := range []federation_key.ActorType{federation_key.FederationHostType, federation_key.FederatedUserType} {
		for _, alg := range []federation_key.Algorithm{federation_key.RsaSha256Cavage, federation_key.RsaSha512Cavage, federation_key.ED25519} {

			var utilAlg util.Algorithm
			switch alg {
			case federation_key.ED25519:
				utilAlg = util.ED25519
			default:
				utilAlg = util.RSA
			}

			key, err := util.RandomPKIXPublicKey(alg.MinKeyLength(), utilAlg)
			require.NoError(t, err)

			sut, err := federation_key.NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				actorType,
				alg,
			)
			require.NoError(t, err, "sut should be valid: %v", alg)
			require.Equal(t, sut.KeyID.String(), keyID)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				Key:       key,
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			_, err = validation.IsValid(sut)
			require.NoError(t, err, "sut should be valid")
			require.Equal(t, sut.KeyID.String(), keyID)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				"",
				key,
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", sut)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				Key:       key,
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err := validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				keyID,
				[]byte{},
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				keyID,
				[]byte{0xb, 0xa, 0xd},
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				Key:       []byte{0xb, 0xa, 0xd},
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				"invalid,KeyID,URL",
				key,
				actorID,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     "invalid,KeyID,URL",
				Key:       key,
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				keyID,
				key,
				-1,
				actorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid actor ID, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				Key:       key,
				ActorID:   -1,
				ActorType: actorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid actor ID, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				federation_key.InvalidActorType,
				alg,
			)
			require.Error(t, err, "sut should be invalid because of invalid actor type, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				Key:       key,
				ActorID:   actorID,
				ActorType: federation_key.InvalidActorType,
				Algorithm: alg,
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid actor type, but was valid %v", res)

			sut, err = federation_key.NewFederationPublicKey(
				ID,
				keyID,
				key,
				actorID,
				actorType,
				federation_key.Algorithm("invalid-algorithm"),
			)
			require.Error(t, err, "sut should be invalid because of invalid signature algorithm, but was valid %v", res)

			sut = &federation_key.FederationPublicKey{
				ID:        ID,
				KeyID:     keyID,
				Key:       key,
				ActorID:   actorID,
				ActorType: actorType,
				Algorithm: federation_key.Algorithm("invalid-algorithm"),
			}
			res, err = validation.IsValid(sut)
			require.Error(t, err, "sut should be invalid because of invalid signature algorithm, but was valid %v", res)
		}
	}
}
