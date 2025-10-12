// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
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

func TestGetFederatedUserByUserID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	localUser, federatedUser, err := GetFederatedUserByUserID(ctx, 42)

	require.NoError(t, err)
	require.NotNil(t, localUser)
	require.NotNil(t, federatedUser)

	assert.Equal(t, "federated-example.net", localUser.LowerName)
	assert.Equal(t, "https://example.net/api/v1/activitypub/user-id/2", federatedUser.ExternalID)

	// This user exists locally but is not federated
	localUser, federatedUser, err = GetFederatedUserByUserID(ctx, 2)

	require.Error(t, err)
	require.Nil(t, localUser)
	require.Nil(t, federatedUser)

	assert.True(t, IsErrUserNotExist(err))

	// The federated user exists but local user is missing
	localUser, federatedUser, err = GetFederatedUserByUserID(ctx, 1337)

	require.Error(t, err)
	require.Nil(t, localUser)
	require.Nil(t, federatedUser)

	assert.ErrorContains(t, err, "FederatedUser table contains entry for user ID 1337, but no user with this ID exists")
}
