// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user_test

import (
	"testing"

	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/require"
)

const (
	valid   = true
	invalid = false
)

func testHostScheme(isValid bool) string {
	if isValid && !setting.Federation.UseInsecureHTTP {
		return "https"
	} else if !isValid && setting.Federation.UseInsecureHTTP {
		return "ftp"
	}
	return "http"
}

func Test_FederatedUserValidation(t *testing.T) {
	hostScheme := testHostScheme(valid)

	sut := user.FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = user.FederatedUser{
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid")
	}
}

func Test_FederatedUserKeyIDValidation(t *testing.T) {
	hostScheme := testHostScheme(valid)

	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: hostScheme,
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID(hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.NoError(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationNilHost(t *testing.T) {
	hostScheme := testHostScheme(valid)

	keyID, err := federation_key.NewKeyID(hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, nil))
}

func Test_FederatedUserKeyIDValidationBadHost(t *testing.T) {
	hostScheme := testHostScheme(valid)

	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: hostScheme,
		HostFqdn:   "bad.host",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID(hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, nil))

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadHostPort(t *testing.T) {
	hostScheme := testHostScheme(valid)

	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: hostScheme,
		HostFqdn:   "bad.host",
		HostPort:   3001,
	}

	keyID, err := federation_key.NewKeyID(hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadScheme(t *testing.T) {
	badHostScheme := testHostScheme(invalid)
	hostScheme := testHostScheme(valid)

	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: hostScheme,
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID := federation_key.KeyID(badHostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))

	keyID = federation_key.KeyID(badHostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	host.HostSchema = badHostScheme

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadHostID(t *testing.T) {
	hostScheme := testHostScheme(valid)

	host := &forgefed.FederationHost{
		ID:         2,
		HostSchema: hostScheme,
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID(hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := user.FederatedUser{
		UserID:                12,
		ExternalID:            "12",
		FederationHostID:      1,
		InboxPath:             "/api/v1/activitypub/user-id/12/inbox",
		NormalizedOriginalURL: hostScheme + "://localhost:3000/api/v1/activitypub/user-id/12",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}
