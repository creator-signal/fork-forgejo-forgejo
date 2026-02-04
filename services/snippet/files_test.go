// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet_test

import (
	"testing"

	api "forgejo.org/modules/structs"
	snippet_service "forgejo.org/services/snippet"

	"github.com/stretchr/testify/assert"
)

func TestSnippetFilesContains(t *testing.T) {
	files := make(snippet_service.SnippetFiles, 2)

	files[0] = &api.SnippetFile{Name: "a.txt"}
	files[1] = &api.SnippetFile{Name: "b.txt"}

	assert.True(t, files.Contains("A.txt"))
	assert.False(t, files.Contains("C.txt"))
}
