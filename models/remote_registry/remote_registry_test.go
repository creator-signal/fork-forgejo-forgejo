// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remote_registry

import (
	"testing"

	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RemoteRegistryValidation(t *testing.T) {
	sut := RemoteRegistry{
		ID:         int64(1),
		Name:       "rr",
		OwnerType:  RRUser,
		OwnerID:    int64(10),
		RemoteURL:  "https://codeberg.org",
		RemoteType: packages.Type("container"),
	}

	if ok, err := validation.IsValid(sut); !ok {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}
}

func Test_CreateRemoteRegistry(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	name := "testreg"
	remoteURL := "https://example.com"
	remoteType := packages.TypeContainer

	rr := RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)

	require.NoError(t, err)

	retrieved := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: "testreg"})
	assert.Equal(t, remoteType, retrieved.RemoteType)
}

func Test_FindRemoteRegistryByName(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	name := "testreg"
	remoteURL := "https://example.com"
	remoteType := packages.TypeContainer

	rr := RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)

	require.NoError(t, err)

	retrieved, err := GetRemoteRegistryByName(t.Context(), RemoteRegistryOwnerType("org"), int64(1), "testreg")
	require.NoError(t, err)
	assert.Equal(t, name, retrieved.Name)
}

func Test_FindRemoteRegistryByOwnerType(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	name := "testreg"
	remoteURL := "https://example.com"
	remoteType := packages.TypeContainer

	rr := RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	name2 := "testreg2"
	remoteURL2 := "https://example.com"
	remoteType2 := packages.TypeContainer
	rr2 := RemoteRegistry{
		Name:       name2,
		RemoteURL:  remoteURL2,
		RemoteType: remoteType2,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)
	require.NoError(t, err)
	err = CreateRemoteRegistry(t.Context(), rr2)
	require.NoError(t, err)

	retrieved, err := GetRemoteRegistriesByOwnerType(t.Context(), RemoteRegistryOwnerType("org"), int64(1))
	require.NoError(t, err)
	assert.Equal(t, name, retrieved[0].Name)
	assert.Equal(t, name2, retrieved[1].Name)
}
