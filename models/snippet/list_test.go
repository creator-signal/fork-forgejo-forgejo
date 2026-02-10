// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet_test

import (
	"testing"

	"forgejo.org/models/db"
	snippet_model "forgejo.org/models/snippet"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnippetListLoadOwner(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	snippetList := make(snippet_model.SnippetList, 4)
	snippetList[0] = unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 1})
	snippetList[1] = unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 2})
	snippetList[2] = unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 3})
	snippetList[3] = unittest.AssertExistsAndLoadBean(t, &snippet_model.Snippet{ID: 4})

	for _, snippet := range snippetList {
		assert.Nil(t, snippet.Owner)
	}

	require.NoError(t, snippetList.LoadOwner(db.DefaultContext))

	assert.Equal(t, int64(2), snippetList[0].Owner.ID)
	assert.Equal(t, int64(2), snippetList[1].Owner.ID)
	assert.Equal(t, int64(2), snippetList[2].Owner.ID)
	assert.Equal(t, int64(3), snippetList[3].Owner.ID)
}
