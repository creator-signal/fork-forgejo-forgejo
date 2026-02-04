// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet_test

import (
	"testing"

	"forgejo.org/models/db"
	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchSnippets(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})

	t.Run("AllGuest", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		snippets, count, err := snippet_model.SearchSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{})
		require.NoError(t, err)

		assert.Len(t, snippets, 2)
		assert.Equal(t, int64(2), count)
		assert.Equal(t, int64(1), snippets[0].ID)
		assert.Equal(t, int64(4), snippets[1].ID)
	})

	t.Run("AllUser", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		snippets, count, err := snippet_model.SearchSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{Actor: user})
		require.NoError(t, err)

		assert.Len(t, snippets, 4)
		assert.Equal(t, int64(4), count)
		assert.Equal(t, int64(1), snippets[0].ID)
		assert.Equal(t, int64(2), snippets[1].ID)
		assert.Equal(t, int64(3), snippets[2].ID)
		assert.Equal(t, int64(4), snippets[3].ID)
	})

	t.Run("AllAdmin", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		snippets, count, err := snippet_model.SearchSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{Actor: admin})
		require.NoError(t, err)

		assert.Len(t, snippets, 4)
		assert.Equal(t, int64(4), count)
		assert.Equal(t, int64(1), snippets[0].ID)
		assert.Equal(t, int64(2), snippets[1].ID)
		assert.Equal(t, int64(3), snippets[2].ID)
		assert.Equal(t, int64(4), snippets[3].ID)
	})

	t.Run("OwnerID", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		snippets, count, err := snippet_model.SearchSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{OwnerID: 2})
		require.NoError(t, err)

		assert.Len(t, snippets, 1)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(1), snippets[0].ID)
	})

	t.Run("Keyword", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		snippets, count, err := snippet_model.SearchSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{Keyword: "another"})
		require.NoError(t, err)

		assert.Len(t, snippets, 1)
		assert.Equal(t, int64(1), count)
		assert.Equal(t, int64(4), snippets[0].ID)
	})
}

func TestCountSnippet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	count, err := snippet_model.CountSnippets(db.DefaultContext, &snippet_model.SearchSnippetsOptions{})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}
