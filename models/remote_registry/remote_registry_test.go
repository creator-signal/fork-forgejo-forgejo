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

func Test_CreateUpdateGetDeleteRemoteRegistry(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	name := "testreg"
	name2 := "testreg2"
	remoteURL := "https://example.com"
	remoteURL2 := "https://registry.example.com"
	remoteType := packages.TypeContainer

	rr := RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteHost: "example.com",
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	rr2 := RemoteRegistry{
		Name:       name2,
		RemoteURL:  remoteURL2,
		RemoteHost: "registry.example.com",
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)
	require.NoError(t, err)
	retrieved := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: name})
	assert.Equal(t, remoteType, retrieved.RemoteType)

	err = UpdateRemoteRegistry(t.Context(), rr2, rr.Name)
	require.NoError(t, err)
	retrieved = unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: name2})
	assert.Equal(t, remoteType, retrieved.RemoteType)

	rrs, err := GetRemoteRegistriesByOwnerType(t.Context(), RROrg, int64(1))
	require.NoError(t, err)
	assert.Equal(t, retrieved.ID, rrs[0].ID)

	err = DeleteRemoteRegistry(t.Context(), RROrg, int64(1), rr2.Name)
	require.NoError(t, err)
	unittest.AssertNotExistsBean(t, &RemoteRegistry{Name: name2})
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
		RemoteHost: "example.com",
		RemotePort: 443,
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
		RemoteHost: "example.com",
		RemotePort: 443,
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
