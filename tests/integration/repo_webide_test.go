// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// enableWebIDEUnit turns on the TypeWebIDE unit for a repository (it is off by default).
func enableWebIDEUnit(t *testing.T, repo *repo_model.Repository) {
	t.Helper()
	require.NoError(t, repo_service.UpdateRepositoryUnits(db.DefaultContext, repo,
		[]repo_model.RepoUnit{{RepoID: repo.ID, Type: unit_model.TypeWebIDE, Config: new(repo_model.UnitConfig)}},
		nil))
}

func TestWebIDE(t *testing.T) {
	// onApplicationRun starts a real server so the commit path's pre-receive hook
	// can call back into Forgejo (git push would otherwise be rejected).
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		// Normal state for most subtests: instance-enabled, generous editable size.
		defer test.MockVariableValue(&setting.Repository.WebIDE.Enabled, true)()
		defer test.MockVariableValue(&setting.Repository.WebIDE.MaxEditableSize, int64(1<<20))()

		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		session := loginUser(t, "user2")

		t.Run("404 when instance setting disabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			defer test.MockVariableValue(&setting.Repository.WebIDE.Enabled, false)()

			req := NewRequest(t, "GET", "/user2/repo1/_ide/tree")
			session.MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("404 when repo unit not enabled", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			// repo1 has no TypeWebIDE unit yet.
			req := NewRequest(t, "GET", "/user2/repo1/_ide/tree")
			session.MakeRequest(t, req, http.StatusNotFound)
		})

		enableWebIDEUnit(t, repo)

		t.Run("shell sets a route-scoped CSP", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "GET", "/user2/repo1/ide")
			resp := session.MakeRequest(t, req, http.StatusOK)
			csp := resp.Header().Get("Content-Security-Policy")
			assert.Contains(t, csp, "worker-src 'self' blob:")
			assert.Contains(t, csp, "'wasm-unsafe-eval'")
			assert.Contains(t, csp, "connect-src 'self'")
		})

		t.Run("tree lists directory entries", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "GET", "/user2/repo1/_ide/tree?ref=master")
			resp := session.MakeRequest(t, req, http.StatusOK)

			var entries []struct {
				Name string `json:"name"`
				Path string `json:"path"`
				Type string `json:"type"`
				Size int64  `json:"size"`
			}
			DecodeJSON(t, resp, &entries)
			require.NotEmpty(t, entries)

			names := make([]string, 0, len(entries))
			for _, e := range entries {
				names = append(names, e.Name)
			}
			assert.Contains(t, names, "README.md")
		})

		t.Run("blob returns base64 editable content", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequest(t, "GET", "/user2/repo1/_ide/blob?ref=master&path=README.md")
			resp := session.MakeRequest(t, req, http.StatusOK)

			var blob struct {
				Path     string `json:"path"`
				Encoding string `json:"encoding"`
				Content  string `json:"content"`
				Editable bool   `json:"editable"`
			}
			DecodeJSON(t, resp, &blob)
			assert.True(t, blob.Editable)
			assert.Equal(t, "base64", blob.Encoding)
			decoded, err := base64.StdEncoding.DecodeString(blob.Content)
			require.NoError(t, err)
			assert.NotEmpty(t, decoded)
		})

		t.Run("blob rejects .git traversal", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			// cleanIDEPath strips a .git component, leaving an empty path -> 404.
			req := NewRequest(t, "GET", "/user2/repo1/_ide/blob?ref=master&path=.git/config")
			session.MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("commit creates a file and it becomes readable", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			body := map[string]any{
				"branch":  "master",
				"message": "add file via Web IDE test",
				"files": []map[string]any{{
					"operation": "create",
					"path":      "webide-test.txt",
					"content":   base64.StdEncoding.EncodeToString([]byte("hello web ide\n")),
				}},
			}
			req := NewRequestWithJSON(t, "POST", "/user2/repo1/_ide/commit", body)
			session.MakeRequest(t, req, http.StatusCreated)

			req = NewRequest(t, "GET", "/user2/repo1/_ide/blob?ref=master&path=webide-test.txt")
			resp := session.MakeRequest(t, req, http.StatusOK)
			var blob struct {
				Content string `json:"content"`
			}
			DecodeJSON(t, resp, &blob)
			decoded, err := base64.StdEncoding.DecodeString(blob.Content)
			require.NoError(t, err)
			assert.Equal(t, "hello web ide\n", string(decoded))
		})

		t.Run("commit with no files is rejected", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			req := NewRequestWithJSON(t, "POST", "/user2/repo1/_ide/commit", map[string]any{
				"branch":  "master",
				"message": "empty",
				"files":   []map[string]any{},
			})
			session.MakeRequest(t, req, http.StatusUnprocessableEntity)
		})
	})
}
