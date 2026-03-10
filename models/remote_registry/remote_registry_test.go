// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package remote_registry

import (
	"testing"

	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/validation"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	regName        string        = "someRegistry"
	remoteURL      string        = "https://registry.example.com"
	remoteHost     string        = "registry.example.com"
	remotePassword string        = "somePassword"
	remoteToken    string        = "someToken"
	remoteType     packages.Type = packages.TypeContainer
)

func Test_RemoteRegistryValidation(t *testing.T) {
	sut := RemoteRegistry{
		ID:         int64(1),
		Name:       regName,
		OwnerType:  RRUser,
		OwnerID:    int64(10),
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
	}

	if ok, err := validation.IsValid(sut); !ok {
		t.Errorf("sut should be valid, %v, %v", sut, err)
	}
}

func Test_CreateUpdateGetDeleteRemoteRegistry(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	name2 := "testreg2"
	remoteURL2 := "https://registry.example.com"
	remoteHost2 := "registry.example.com"

	rr := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteHost: remoteHost,
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	rr2 := RemoteRegistry{
		Name:       name2,
		RemoteURL:  remoteURL2,
		RemoteHost: remoteHost2,
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)
	require.NoError(t, err)
	retrieved := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: regName})
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

func Test_CanCreateForUserAndOrg(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	rr := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteHost: remoteHost,
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RRUser,
		OwnerID:    int64(1),
	}

	rr2 := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteHost: remoteHost,
		RemotePort: 443,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(2),
	}

	err := CreateRemoteRegistry(t.Context(), rr)
	require.NoError(t, err)
	retrieved := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: rr.Name, OwnerType: rr.OwnerType, OwnerID: rr.OwnerID})
	assert.Equal(t, remoteType, retrieved.RemoteType)

	rrs, err := GetRemoteRegistriesByOwnerType(t.Context(), rr.OwnerType, rr.OwnerID)
	require.NoError(t, err)
	assert.Equal(t, retrieved.ID, rrs[0].ID)

	err = CreateRemoteRegistry(t.Context(), rr2)
	require.NoError(t, err)
	retrieved2 := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{Name: rr2.Name, OwnerType: rr2.OwnerType, OwnerID: rr2.OwnerID})
	assert.Equal(t, remoteType, retrieved2.RemoteType)

	rrs2, err := GetRemoteRegistriesByOwnerType(t.Context(), rr2.OwnerType, rr2.OwnerID)
	require.NoError(t, err)
	assert.Equal(t, retrieved2.ID, rrs2[0].ID)
}

func Test_SetGetCredentials(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	rr := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		RemoteHost: remoteHost,
		RemotePort: 443,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	rr.SetRemotePassword(remotePassword)
	rr.SetRemoteToken(remoteToken)

	pw, err := rr.GetRemotePassword()
	require.NoError(t, err)
	tk, err := rr.GetRemoteToken()
	require.NoError(t, err)

	assert.Equal(t, remotePassword, pw)
	assert.Equal(t, remoteToken, tk)
}

func Test_FindRemoteRegistryByName(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	rr := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		RemoteHost: remoteHost,
		RemotePort: 443,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)

	require.NoError(t, err)

	retrieved, err := GetRemoteRegistryByName(t.Context(), RemoteRegistryOwnerType("org"), int64(1), regName)
	require.NoError(t, err)
	assert.Equal(t, regName, retrieved.Name)
}

func Test_FindRemoteRegistryByOwnerType(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	rr := RemoteRegistry{
		Name:       regName,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		RemoteHost: remoteHost,
		RemotePort: 443,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	name2 := "testreg2"
	remoteURL2 := "https://example.com"
	rr2 := RemoteRegistry{
		Name:       name2,
		RemoteURL:  remoteURL2,
		RemoteType: remoteType,
		OwnerType:  RROrg,
		OwnerID:    int64(1),
	}

	err := CreateRemoteRegistry(t.Context(), rr)
	require.NoError(t, err)
	err = CreateRemoteRegistry(t.Context(), rr2)
	require.NoError(t, err)

	retrieved, err := GetRemoteRegistriesByOwnerType(t.Context(), RemoteRegistryOwnerType("org"), int64(1))
	require.NoError(t, err)
	assert.Equal(t, regName, retrieved[0].Name)
	assert.Equal(t, name2, retrieved[1].Name)
}
