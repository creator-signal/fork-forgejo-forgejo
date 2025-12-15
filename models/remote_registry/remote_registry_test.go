// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package remote_registry

import (
	"testing"

	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"
	"forgejo.org/modules/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RemoteRegistryValidation(t *testing.T) {
	sut := RemoteRegistry{
		ID:          int64(1),
		Name:        "rr",
		OwnerType:   RRUser,
		OwnerID:     int64(10),
		RemoteURL:   "https://codeberg.org",
		RemoteType:  packages.Type("container"),
		CreatedUnix: timeutil.TimeStampNow(),
		UpdatedUnix: timeutil.TimeStampNow(),
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
	opts := RROpts{
		RemoteRegistryOwnerType("org"),
		int64(1),
		RRCredentials{},
	}
	rr, err := NewRemoteRegistry(name, remoteURL, remoteType, opts)

	require.NoError(t, err)

	err = CreateRemoteRegistry(t.Context(), rr)

	require.NoError(t, err)

	retrieved := unittest.AssertExistsAndLoadBean(t, &RemoteRegistry{ID: 1})
	assert.Equal(t, remoteType, retrieved.RemoteType)
}
