// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webfinger_test

import (
	"testing"

	"forgejo.org/modules/webfinger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildURL(t *testing.T) {
	// valid URL actor, I know it's valid because it's me!
	url, err := webfinger.BuildURL("@famfo@frogs.lgbt")
	require.NoError(t, err)
	assert.Equal(t, "https://frogs.lgbt/.well-known/webfinger?resource=acct%3Afamfo%40frogs.lgbt", url.String())

	// same, valid actor without leading @
	url, err = webfinger.BuildURL("famfo@frogs.lgbt")
	require.NoError(t, err)
	assert.Equal(t, "https://frogs.lgbt/.well-known/webfinger?resource=acct%3Afamfo%40frogs.lgbt", url.String())

	// internationalized domains
	url, err = webfinger.BuildURL("@user@ευ.ευ")
	require.NoError(t, err)
	assert.Equal(t, "https://xn--qxa6a.xn--qxa6a/.well-known/webfinger?resource=acct%3Auser%40xn--qxa6a.xn--qxa6a", url.String())

	// Custom port
	url, err = webfinger.BuildURL("@user@example.org:1337")
	require.NoError(t, err)
	assert.Equal(t, "https://example.org:1337/.well-known/webfinger?resource=acct%3Auser%40example.org%3A1337", url.String())

	// "Reserved" seperators by go's url.PathEscape
	url, err = webfinger.BuildURL("@user;name@example.com")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/.well-known/webfinger?resource=acct%3Auser%3Bname%40example.com", url.String())
}
