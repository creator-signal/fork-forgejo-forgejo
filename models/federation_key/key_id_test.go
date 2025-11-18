// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key_test

import (
	"testing"

	"forgejo.org/models/federation_key"
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/require"
)

const (
	keyIDFragment      = "https://forgejo.org/api/v1/activitypub/actor/1#meow"
	keyIDQuery         = "https://forgejo.org/api/v1/activitypub/actor/1?key-id=meow"
	keyIDFragmentQuery = "https://forgejo.org/api/v1/activitypub/actor/1#meow?algorithm=rsa-v1_5-sha256"
)

func Test_FederationKeyID(t *testing.T) {
	var err error

	for _, keyID := range []string{keyIDFragment, keyIDQuery, keyIDFragmentQuery} {
		_, err = federation_key.NewKeyID(keyID)
		require.NoError(t, err, "expected valid key ID, error occured: %v", err)

		_, err = validation.IsValid(federation_key.KeyID(ap.IRI(keyID)))
		require.NoError(t, err, "expected valid key ID, error occured: %v", err)
	}

	_, err = federation_key.NewKeyID("invalid.url?%^")
	require.Error(t, err, "expected error for invalid URL")

	_, err = federation_key.NewKeyID("gopher://forgejo.org/api/v1/activitypub/actor/1#meow")
	require.Error(t, err, "expected error for invalid scheme")

	_, err = federation_key.NewKeyID("https://user@forgejo.org/api/v1/activitypub/actor/1#meow")
	require.Error(t, err, "expected error for including insecure username")

	_, err = federation_key.NewKeyID("https://user:password@forgejo.org/api/v1/activitypub/actor/1#meow")
	require.Error(t, err, "expected error for including insecure username + password")

	_, err = federation_key.NewKeyID("https://forgejo.org/../api/v1/activitypub/actor/1#meow")
	require.Error(t, err, "expected error for relative path, possible path traversal")
}
