// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet_test

import (
	"testing"

	api "forgejo.org/modules/structs"
	snippet_service "forgejo.org/services/snippet"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnippetFilesContains(t *testing.T) {
	files := make(snippet_service.SnippetFiles, 2)

	files[0] = &api.SnippetFile{Name: "a.txt"}
	files[1] = &api.SnippetFile{Name: "b.txt"}

	assert.True(t, files.Contains("A.txt"))
	assert.False(t, files.Contains("C.txt"))
}

func TestSnippetFilesValidateNames(t *testing.T) {
	files := make(snippet_service.SnippetFiles, 1)

	files[0] = &api.SnippetFile{Name: "test.txt"}
	require.NoError(t, files.Validate())

	files[0] = &api.SnippetFile{Name: "dir/test.txt"}
	assert.Error(t, files.Validate())
}

func TestSnippetFilesGetLanguage(t *testing.T) {
	files := make(snippet_service.SnippetFiles, 1)

	files[0] = &api.SnippetFile{Name: "test.py", Content: "print()"}

	assert.Equal(t, "Python", files.GetLanguage())
}
