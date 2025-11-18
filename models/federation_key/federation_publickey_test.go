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

	key, err := util.RandomPKIXPublicKey(3072)
	require.NoError(t, err)

	sut, err := NewFederationPublicKey(
		ID,
		keyID,
		key,
	)
	require.NoError(t, err, "sut should be valid")
	require.Equal(t, sut.KeyID.String(), keyID)

	sut = &FederationPublicKey{
		ID:    ID,
		KeyID: keyID,
		Key:   key,
	}
	_, err = validation.IsValid(sut)
	require.NoError(t, err, "sut should be valid")
	require.Equal(t, sut.KeyID.String(), keyID)

	sut, err = NewFederationPublicKey(
		ID,
		"",
		key,
	)
	require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", sut)

	sut = &FederationPublicKey{
		ID:  ID,
		Key: key,
	}
	res, err := validation.IsValid(sut)
	require.Error(t, err, "sut should be invalid because of NULL key ID, but was valid %v", res)

	sut, err = NewFederationPublicKey(
		ID,
		keyID,
		[]byte{},
	)
	require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

	sut = &FederationPublicKey{
		ID:    ID,
		KeyID: keyID,
	}
	res, err = validation.IsValid(sut)
	require.Error(t, err, "sut should be invalid because of NULL key, but was valid %v", res)

	sut, err = NewFederationPublicKey(
		ID,
		keyID,
		[]byte{0xb, 0xa, 0xd},
	)
	require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

	sut = &FederationPublicKey{
		ID:    ID,
		KeyID: keyID,
		Key:   []byte{0xb, 0xa, 0xd},
	}
	res, err = validation.IsValid(sut)
	require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

	sut, err = NewFederationPublicKey(
		ID,
		"invalid,KeyID,URL",
		key,
	)
	require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)

	sut = &FederationPublicKey{
		ID:    ID,
		KeyID: "invalid,KeyID,URL",
		Key:   key,
	}
	res, err = validation.IsValid(sut)
	require.Error(t, err, "sut should be invalid because of invalid key, but was valid %v", res)
}
