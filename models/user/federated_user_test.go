// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"fmt"
	"strconv"
	"testing"

	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/require"
)

func Test_FederatedUserValidation(t *testing.T) {
	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FederatedUser{
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid")
	}
}

func Test_FederatedUserKeyIDValidation(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: "http",
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.NoError(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationNilHost(t *testing.T) {
	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, nil))
}

func Test_FederatedUserKeyIDValidationBadHost(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: "http",
		HostFqdn:   "bad.host",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, nil))

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadHostPort(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: "http",
		HostFqdn:   "bad.host",
		HostPort:   3001,
	}

	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadScheme(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: "https",
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))

	keyID, err = federation_key.NewKeyID("https://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)
	host.HostSchema = "http"

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadHostID(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         2,
		HostSchema: "http",
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	keyID, err := federation_key.NewKeyID("http://localhost:3000/api/v1/activitypub/user-id/12#main-key")
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		InboxPath:        "/api/v1/activitypub/user-id/12/inbox",
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}

func Test_FederatedUserKeyIDValidationBadUserID(t *testing.T) {
	host := &forgefed.FederationHost{
		ID:         1,
		HostSchema: "http",
		HostFqdn:   "localhost",
		HostPort:   3000,
	}

	userID := 12

	keyID, err := federation_key.NewKeyID(fmt.Sprintf("http://localhost:3000/api/v1/activitypub/user-id/%v#main-key", userID+1))
	require.NoError(t, err)

	sut := FederatedUser{
		UserID:           int64(userID),
		ExternalID:       strconv.Itoa(userID),
		FederationHostID: 1,
		InboxPath:        fmt.Sprintf("/api/v1/activitypub/user-id/%v/inbox", userID),
	}

	require.Error(t, sut.ValidateKeyIDWithHost(keyID, host))
}
