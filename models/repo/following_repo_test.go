// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_FollowingRepoValidation(t *testing.T) {
	sut := FollowingRepo{
		RepoID:           12,
		ExternalID:       "12",
		FederationHostID: 1,
		URI:              "http://localhost:3000/api/v1/activitypub/repo-id/1",
	}
	if res, err := validation.IsValid(sut); !res {
		t.Errorf("sut should be valid but was %q", err)
	}

	sut = FollowingRepo{
		ExternalID:       "12",
		FederationHostID: 1,
		URI:              "http://localhost:3000/api/v1/activitypub/repo-id/1",
	}
	if res, _ := validation.IsValid(sut); res {
		t.Error("sut should be invalid")
	}
}

func TestGetFollowingRepoById(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ctx := t.Context()

	followingRepo, err := GetFollowingRepoByID(ctx, 1)

	require.NoError(t, err)
	require.NotNil(t, followingRepo)

	assert.Equal(t, int64(1), followingRepo.ID)
	assert.Equal(t, "https://forge.example.com/api/v1/activitypub/repository-id/1", followingRepo.URI)

	followingRepo, err = GetFollowingRepoByID(ctx, 1337)

	require.Error(t, err)
	require.Nil(t, followingRepo)

	assert.True(t, IsErrRepoNotExist(err))
}
