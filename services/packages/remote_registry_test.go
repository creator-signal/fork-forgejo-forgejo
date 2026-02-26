// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package packages

import (
	"testing"

	"forgejo.org/models/packages"
	remote_registry_model "forgejo.org/models/remote_registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewRemoteRegistry(t *testing.T) {
	name := "testreg"
	remoteURL := "https://example.com"
	remoteType := packages.TypeContainer
	opts := RROpts{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		OwnerType:  remote_registry_model.RemoteRegistryOwnerType("org"),
		OwnerID:    int64(1),
		Auth: RRCredentials{
			RemoteUser:     "someUser",
			RemotePassword: "somePassword",
			RemoteToken:    "someToken",
		},
	}

	rr, err := NewRemoteRegistry(opts)
	require.NoError(t, err)

	assert.Equal(t, name, rr.Name)
	assert.Equal(t, remoteURL, rr.RemoteURL)
	assert.Equal(t, remoteType, rr.RemoteType)
	assert.Equal(t, opts.OwnerType, rr.OwnerType)
}

func Test_NewRemoteRegistryInvalid(t *testing.T) {
	name := "tes,treg"
	remoteURL := "https://example.com"
	remoteType := packages.TypeContainer
	opts := RROpts{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteType: remoteType,
		OwnerType:  remote_registry_model.RemoteRegistryOwnerType("org"),
		OwnerID:    int64(1),
		Auth:       RRCredentials{},
	}

	_, err := NewRemoteRegistry(opts)

	assert.Error(t, err)
}
