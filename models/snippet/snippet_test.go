// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"forgejo.org/models/db"
	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnippetVisibilityFromName(t *testing.T) {
	visibility, err := snippet_model.SnippetVisibilityFromName("public")
	require.NoError(t, err)
	assert.Equal(t, snippet_model.SnippetVisibilityPublic, visibility)

	visibility, err = snippet_model.SnippetVisibilityFromName("hidden")
	require.NoError(t, err)
	assert.Equal(t, snippet_model.SnippetVisibilityHidden, visibility)

	visibility, err = snippet_model.SnippetVisibilityFromName("private")
	require.NoError(t, err)
	assert.Equal(t, snippet_model.SnippetVisibilityPrivate, visibility)

	_, err = snippet_model.SnippetVisibilityFromName("invalid")
	require.Error(t, err)
}

func TestGetSnippetByUUID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippet, err := snippet_model.GetSnippetByUUID(db.DefaultContext, "df852aec")
	require.NoError(t, err)
	assert.Equal(t, int64(1), snippet.ID)

	snippet, err = snippet_model.GetSnippetByUUID(db.DefaultContext, "invalid")
	require.Error(t, err)
	assert.True(t, snippet_model.IsErrSnippetNotExist(err))
	assert.Nil(t, snippet)
}

func TestCountOwnerSnippets(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	count, err := snippet_model.CountOwnerSnippets(db.DefaultContext, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestSnippetGetRepoPath(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})

	assert.Equal(t, filepath.Join(setting.Snippet.RootPath, "df852aec.git"), snippet.GetRepoPath())
}

func TestSnippetLink(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})

	assert.Equal(t, "/snippets/df852aec", snippet.Link())
}

func TestSnippetHTMLURL(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippets := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})

	assert.Equal(t, fmt.Sprintf("%ssnippets/df852aec", setting.AppURL), snippets.HTMLURL())
}

func TestSnippetLoadOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})

	assert.Nil(t, snippet.Owner)

	require.NoError(t, snippet.LoadOwner(db.DefaultContext))

	assert.Equal(t, int64(2), snippet.Owner.ID)
}

func TestSnippetIsOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
	snippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})

	assert.False(t, snippet.IsOwner(nil))
	assert.True(t, snippet.IsOwner(user2))
	assert.False(t, snippet.IsOwner(user3))
}

func TestSnippetHasAccess(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	publicSnippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})
	hiddenSnippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 2})
	privateSnippet := unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 3})

	assert.True(t, publicSnippet.HasAccess(nil))
	assert.True(t, publicSnippet.HasAccess(user))

	assert.True(t, hiddenSnippet.HasAccess(nil))
	assert.True(t, hiddenSnippet.HasAccess(user))

	assert.False(t, privateSnippet.HasAccess(nil))
	assert.True(t, privateSnippet.HasAccess(user))
}
