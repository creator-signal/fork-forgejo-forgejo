// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// daemonExportFile returns the full path to the git-daemon-export-ok file
// for a given repo owner and repo name. The repo name is lowercased to match
// the filesystem path used by repo.RepoPath().
func daemonExportFile(ownerName, repoName string) string {
	return filepath.Join(setting.RepoRootPath, ownerName, strings.ToLower(repoName)+".git", "git-daemon-export-ok")
}

// fileExists checks whether a file exists and is not a directory.
func fileExists(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// TestDaemonExportOKRepoCreation verifies that the git-daemon-export-ok
// file is correctly created (or not) during initial repo creation.
func TestDaemonExportOKRepoCreation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
	)

	t.Run("creating public repo with public owner creates export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repoName := "daemon-test-public-" + util.CryptoRandomString(util.RandomStringLow)
		req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
			AutoInit:    true,
			Name:        repoName,
			Private:     false,
			Description: "daemon export test repo",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		path := daemonExportFile("user2", repoName)
		assert.True(t, fileExists(t, path), "git-daemon-export-ok should exist for newly created public repo with public owner")
	})

	t.Run("creating private repo with public owner does not create export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repoName := "daemon-test-private-" + util.CryptoRandomString(util.RandomStringLow)
		req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
			AutoInit:    true,
			Name:        repoName,
			Private:     true,
			Description: "daemon export test repo",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		path := daemonExportFile("user2", repoName)
		assert.False(t, fileExists(t, path), "git-daemon-export-ok should not exist for newly created private repo with public owner")
	})
}

// TestDaemonExportOKOwnerVisibilityPropagation verifies that when a user or
// organization changes visibility, the git-daemon-export-ok file is updated
// for all repos owned by that user/org.
func TestDaemonExportOKOwnerVisibilityPropagation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	adminToken := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteAdmin)
	userToken := getTokenForLoggedInUser(t, loginUser(t, "user2"),
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
	)

	// Create a public repo under user2 to use for propagation testing.
	// This creates the repo through the API which goes through
	// CreateRepositoryByExample, triggering CheckDaemonExportOK.
	repoName := "daemon-propagate-" + util.CryptoRandomString(util.RandomStringLow)
	req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
		AutoInit:    true,
		Name:        repoName,
		Private:     false,
		Description: "daemon propagation test repo",
	}).AddTokenAuth(userToken)
	MakeRequest(t, req, http.StatusCreated)

	repoPath := daemonExportFile("user2", repoName)

	t.Run("newly created public repo has export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		assert.True(t, fileExists(t, repoPath), "precondition: export file should exist for public repo with public owner")
	})

	t.Run("changing user visibility to private removes export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		urlStr := fmt.Sprintf("/api/v1/admin/users/%s", "user2")
		req := NewRequestWithJSON(t, "PATCH", urlStr, api.EditUserOption{
			Visibility: "private",
		}).AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusOK)

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		assert.True(t, user2.Visibility.IsPrivate(), "user2 should now have private visibility")

		assert.False(t, fileExists(t, repoPath),
			"git-daemon-export-ok should be removed when owner becomes private")
	})
}

// TestDaemonExportOKOrgVisibilityPropagation verifies that org visibility
// changes propagate daemon export files to all owned repos.
func TestDaemonExportOKOrgVisibilityPropagation(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2Session := loginUser(t, "user2")
	orgToken := getTokenForLoggedInUser(t, user2Session,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteOrganization,
		auth_model.AccessTokenScopeWriteUser,
	)

	// Create a public repo under org3 (which is public).
	repoName := "daemon-org-prop-" + util.CryptoRandomString(util.RandomStringLow)
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/orgs/org3/repos"), &api.CreateRepoOption{
		AutoInit:    true,
		Name:        repoName,
		Private:     false,
		Description: "daemon org propagation test repo",
	}).AddTokenAuth(orgToken)
	MakeRequest(t, req, http.StatusCreated)

	repoPath := daemonExportFile("org3", repoName)

	t.Run("newly created public org repo has export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		assert.True(t, fileExists(t, repoPath), "precondition: export file should exist for public org repo")
	})

	t.Run("changing org visibility to private removes export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequestWithJSON(t, "PATCH", "/api/v1/orgs/org3", api.EditOrgOption{
			Visibility: "private",
		}).AddTokenAuth(orgToken)
		MakeRequest(t, req, http.StatusOK)

		org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
		assert.True(t, org3.Visibility.IsPrivate(), "org3 should now have private visibility")

		assert.False(t, fileExists(t, repoPath),
			"git-daemon-export-ok should be removed from org repo when org becomes private")
	})
}

// TestDaemonExportOKVisibilityChangeDoesNotAffectPrivateRepos verifies that
// when user visibility changes, private repos remain without export file.
func TestDaemonExportOKVisibilityChangeDoesNotAffectPrivateRepos(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	adminToken := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteAdmin)
	userToken := getTokenForLoggedInUser(t, loginUser(t, "user2"),
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
	)

	// Create a private repo under user2.
	repoName := "daemon-private-prop-" + util.CryptoRandomString(util.RandomStringLow)
	req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
		AutoInit:    true,
		Name:        repoName,
		Private:     true,
		Description: "daemon private propagation test repo",
	}).AddTokenAuth(userToken)
	MakeRequest(t, req, http.StatusCreated)

	repoPath := daemonExportFile("user2", repoName)

	t.Run("private repo has no export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()
		assert.False(t, fileExists(t, repoPath), "precondition: export file should not exist for private repo")
	})

	t.Run("changing owner to private does not create export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		urlStr := fmt.Sprintf("/api/v1/admin/users/%s", "user2")
		req := NewRequestWithJSON(t, "PATCH", urlStr, api.EditUserOption{
			Visibility: "private",
		}).AddTokenAuth(adminToken)
		MakeRequest(t, req, http.StatusOK)

		assert.False(t, fileExists(t, repoPath),
			"git-daemon-export-ok should not exist for private repo when owner is private")
	})
}

// TestDaemonExportOKRepoCreationWithPrivateOwner verifies that creating a repo
// under a private owner never creates an export file, even for public repos.
func TestDaemonExportOKRepoCreationWithPrivateOwner(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// First, make user2 private
	adminToken := getUserToken(t, "user1", auth_model.AccessTokenScopeWriteAdmin)
	urlStr := fmt.Sprintf("/api/v1/admin/users/%s", "user2")
	req := NewRequestWithJSON(t, "PATCH", urlStr, api.EditUserOption{
		Visibility: "private",
	}).AddTokenAuth(adminToken)
	MakeRequest(t, req, http.StatusOK)

	session := loginUser(t, "user2")
	token := getTokenForLoggedInUser(t, session,
		auth_model.AccessTokenScopeWriteRepository,
		auth_model.AccessTokenScopeWriteUser,
	)

	t.Run("public repo with private owner has no export file", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		repoName := "daemon-test-priv-owner-" + util.CryptoRandomString(util.RandomStringLow)
		req := NewRequestWithJSON(t, "POST", "/api/v1/user/repos", &api.CreateRepoOption{
			AutoInit:    true,
			Name:        repoName,
			Private:     false,
			Description: "daemon export test repo",
		}).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusCreated)

		path := daemonExportFile("user2", repoName)
		assert.False(t, fileExists(t, path), "git-daemon-export-ok should not exist for public repo with private owner")
	})
}
