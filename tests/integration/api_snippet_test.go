// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPISnippetsSearch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/snippets/search?sort=alphabetically")
	resp := MakeRequest(t, req, http.StatusOK)

	var snippets api.SnippetList
	DecodeJSON(t, resp, &snippets)

	assert.Len(t, snippets.Snippets, 2)

	assert.Equal(t, int64(4), snippets.Snippets[0].ID)
	assert.Equal(t, int64(3), snippets.Snippets[0].Owner.ID)

	assert.Equal(t, int64(1), snippets.Snippets[1].ID)
	assert.Equal(t, int64(2), snippets.Snippets[1].Owner.ID)
}

func TestAPISnippetsCreate(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteSnippet)

	newSnippet := &api.CreateSnippetOption{
		Name:        "New Snippet",
		Visibility:  "public",
		Description: "New Description",
		Files: []*api.SnippetFile{
			{
				Name:    "new.txt",
				Content: "New text",
			},
		},
	}

	req := NewRequestWithJSON(t, "POST", "/api/v1/snippets", &newSnippet).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var snippet api.Snippet
	DecodeJSON(t, resp, &snippet)

	assert.Equal(t, "New Snippet", snippet.Name)
	assert.Equal(t, "public", snippet.Visibility)
	assert.Equal(t, "New Description", snippet.Description)
	assert.Equal(t, int64(2), snippet.Owner.ID)
}

func TestAPISnippetsGet(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	req := NewRequest(t, "GET", "/api/v1/snippets/df852aec")
	resp := MakeRequest(t, req, http.StatusOK)

	var snippet api.Snippet
	DecodeJSON(t, resp, &snippet)

	assert.Equal(t, "PublicSnippet", snippet.Name)
	assert.Equal(t, "This is a Description", snippet.Description)
	assert.Equal(t, "public", snippet.Visibility)
	assert.Equal(t, int64(2), snippet.Owner.ID)
}

func TestAPISnippetsFiles(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteSnippet)

	req := NewRequest(t, "GET", "/api/v1/snippets/df852aec/files")
	resp := MakeRequest(t, req, http.StatusOK)

	var files []*api.SnippetFile
	DecodeJSON(t, resp, &files)

	assert.Len(t, files, 1)
	assert.Equal(t, "test.txt", files[0].Name)
	assert.Equal(t, "Hello World", files[0].Content)

	newFiles := &api.UpdateSnippetFilesOption{
		Files: []*api.SnippetFile{
			{
				Name:    "new.txt",
				Content: "New text",
			},
		},
	}

	MakeRequest(t, NewRequestWithJSON(t, "POST", "/api/v1/snippets/df852aec/files", &newFiles).AddTokenAuth(token), http.StatusNoContent)

	req = NewRequest(t, "GET", "/api/v1/snippets/df852aec/files")
	resp = MakeRequest(t, req, http.StatusOK)

	DecodeJSON(t, resp, &files)

	assert.Len(t, files, 1)
	assert.Equal(t, "new.txt", files[0].Name)
	assert.Equal(t, "New text", files[0].Content)
}

func TestAPISnippetsDelete(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteSnippet)

	MakeRequest(t, NewRequest(t, "DELETE", "/api/v1/snippets/df852aec").AddTokenAuth(token), http.StatusNoContent)
}
