// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package forge

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_ForgeAccessors(t *testing.T) {
	forge := NewForge()

	url := "URL"
	forge.SetURL(url)
	assert.Equal(t, url, forge.GetURL())

	id := int64(1324)
	forge.SetID(id)
	assert.Equal(t, id, forge.GetID())

	token := "token"
	forge.SetToken(token)
	assert.Equal(t, token, forge.GetToken())

	typ := "TYPE"
	forge.SetType(typ)
	assert.Equal(t, typ, forge.GetType())
}

func TestF3Model_ForgeEncryption(t *testing.T) {
	forge := NewForge()
	forge.SetURL("")
	forge.SetID(1234)

	token := "token"
	forge.SetToken(token)
	encryptedToken := forge.encryptToken()
	forge.SetToken(string(encryptedToken))
	decryptedToken, err := forge.decryptToken()
	require.NoError(t, err)
	assert.Equal(t, token, string(decryptedToken))
}

func TestF3Model_ForgeDatabase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	url := "URL"
	token := "token"
	forge := NewForge()
	forge.SetURL(url)
	forge.SetToken(token)
	typ := "TYPE"
	forge.SetType(typ)

	for range 2 {
		forge, err := Upsert(t.Context(), forge)
		require.NoError(t, err)
		assert.NotZero(t, forge.GetID())
		assert.Equal(t, url, forge.GetURL())
		assert.Equal(t, token, forge.GetToken())
		assert.Equal(t, typ, forge.GetType())
	}

	{
		noNewToken := NewForge()
		noNewToken.SetURL("URL")
		noNewToken.SetType(typ)
		sameForge, err := Upsert(t.Context(), noNewToken)
		require.NoError(t, err)
		assert.True(t, Equal(forge, sameForge))
	}

	{
		forge, err := Get(t.Context(), FindOptions{URL: &url})
		require.NoError(t, err)
		assert.Equal(t, token, forge.GetToken())
	}

	{
		unknownURL := "unknown"
		forge, err := Get(t.Context(), FindOptions{URL: &unknownURL})
		require.NoError(t, err)
		assert.Nil(t, forge)
	}

	{
		forges, err := Find(t.Context(), FindOptions{URL: &url})
		require.NoError(t, err)
		assert.Len(t, forges, 1)
		forge = forges[0]
		assert.Equal(t, token, forge.GetToken())
	}
}
