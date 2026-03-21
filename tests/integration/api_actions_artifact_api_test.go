// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"archive/zip"
	"bytes"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIListActionRunArtifacts(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	// Run 791 has "multi-file-download" artifact (V3, confirmed)
	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts", 791).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result api.ActionArtifactResponse
	DecodeJSON(t, resp, &result)

	assert.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, "multi-file-download", result.Entries[0].Name)
	assert.Equal(t, int64(2048), result.Entries[0].Size)
	assert.Equal(t, "completed", result.Entries[0].Status)
	assert.Contains(t, result.Entries[0].ArchiveDownloadURL, "/api/v1/repos/user5/repo4/actions/runs/791/artifacts/multi-file-download")
	assert.False(t, result.Entries[0].CreatedAt.IsZero())
	assert.False(t, result.Entries[0].ExpiresAt.IsZero())
}

func TestAPIListActionRunArtifacts_V4(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	// Run 792 has "artifact-v4-download" artifact (V4, confirmed)
	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts", 792).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var result api.ActionArtifactResponse
	DecodeJSON(t, resp, &result)

	assert.Equal(t, int64(1), result.TotalCount)
	require.Len(t, result.Entries, 1)
	assert.Equal(t, "artifact-v4-download", result.Entries[0].Name)
	assert.Equal(t, int64(1024), result.Entries[0].Size)
	assert.Equal(t, "completed", result.Entries[0].Status)
}

func TestAPIListActionRunArtifacts_NotFound(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts", 999999).AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIDownloadActionRunArtifact_V3(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 791, "multi-file-download").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	// Verify it's a ZIP
	assert.Contains(t, resp.Header().Get("Content-Disposition"), "multi-file-download.zip")

	// Verify ZIP contents
	body := resp.Body.Bytes()
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	require.NoError(t, err)
	assert.NotEmpty(t, zr.File)
}

func TestAPIDownloadActionRunArtifact_V4(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 792, "artifact-v4-download").AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	assert.Contains(t, resp.Header().Get("Content-Disposition"), "artifact-v4-download.zip")
	// V4 artifact is 1024 bytes of 'D'
	assert.Equal(t, 1024, resp.Body.Len())
}

func TestAPIDownloadActionRunArtifact_NotFound(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeReadRepository)

	req := NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 791, "nonexistent").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIDeleteActionRunArtifact(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeWriteRepository)

	// Delete the artifact
	req := NewRequestf(t, "DELETE", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 791, "multi-file-download").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNoContent)

	// Verify artifact is now pending deletion
	artifacts, err := db.Find[actions_model.ActionArtifact](t.Context(), actions_model.FindArtifactsOptions{
		RunID:        791,
		ArtifactName: "multi-file-download",
	})
	require.NoError(t, err)
	for _, art := range artifacts {
		assert.Equal(t, int64(actions_model.ArtifactStatusPendingDeletion), art.Status)
	}

	// Verify list no longer includes it
	req = NewRequestf(t, "GET", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts", 791).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var listed api.ActionArtifactResponse
	DecodeJSON(t, resp, &listed)
	assert.Empty(t, listed.Entries)
	assert.Equal(t, int64(0), listed.TotalCount)
}

func TestAPIDeleteActionRunArtifact_NotFound(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestf(t, "DELETE", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 791, "nonexistent").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestAPIDeleteActionRunArtifact_Unauthorized(t *testing.T) {
	defer prepareTestEnvActionsArtifacts(t)()

	// user2 doesn't have write access to user5/repo4
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	token := getUserToken(t, user.LowerName, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequestf(t, "DELETE", "/api/v1/repos/user5/repo4/actions/runs/%d/artifacts/%s", 791, "multi-file-download").AddTokenAuth(token)
	MakeRequest(t, req, http.StatusForbidden)
}
