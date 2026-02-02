// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	"net/url"
	"testing"

	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	mock_server "forgejo.org/modules/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ConvertRegClientManifest(t *testing.T) {

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	name := "testreg"
	imageName := "test/test-image"
	remoteURL := server.URL
	url, err := url.Parse(remoteURL)
	require.NoError(t, err)
	host := url.Host
	remoteType := packages.TypeContainer

	rr := rr_model.RemoteRegistry{
		Name:       name,
		RemoteURL:  remoteURL,
		RemoteHost: host,
		RemoteType: remoteType,
		OwnerType:  rr_model.RROrg,
		OwnerID:    int64(1),
	}

	client, err := NewContainerRegistryClient(&rr)

	r, err := client.NewRef(imageName)
	regManifest, err := client.GetManifest(t.Context(), r)

	t.Logf("Ref was: %s", regManifest.GetRef().Reference)
	assert.NotEmpty(t, regManifest.GetRef())

	// Woraus besteht ein Forgejo Manifest?

}
