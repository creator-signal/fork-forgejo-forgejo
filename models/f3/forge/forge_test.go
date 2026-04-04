// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package forge

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/util"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_ForgeDatabase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	url := "URL"
	forge := NewForge()
	forge.SetURL(url)
	typ := "TYPE"
	forge.SetType(typ)

	t.Run("Upsert idempotent", func(t *testing.T) {
		for range 2 {
			forge, err := Upsert(t.Context(), forge)
			require.NoError(t, err)
			assert.NotZero(t, forge.GetID())
			assert.Equal(t, url, forge.GetURL())
			assert.Equal(t, typ, forge.GetType())
		}
	})

	t.Run("Get found", func(t *testing.T) {
		forge, err := Get(t.Context(), FindOptions{URL: &url})
		require.NoError(t, err)
		assert.NotNil(t, forge)
	})

	t.Run("Get not found", func(t *testing.T) {
		unknownURL := "unknown"
		forge, err := Get(t.Context(), FindOptions{URL: &unknownURL})
		require.ErrorIs(t, err, util.ErrNotExist)
		assert.Nil(t, forge)
	})

	t.Run("Find", func(t *testing.T) {
		forges, err := Find(t.Context(), FindOptions{URL: &url})
		require.NoError(t, err)
		assert.Len(t, forges, 1)
		forge = forges[0]
		assert.NotNil(t, forge)
	})
}
