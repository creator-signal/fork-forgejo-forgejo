// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAPIWatch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := "user1"
	repo := "user2/repo1"

	session := loginUser(t, user)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadUser)
	tokenWithRepoScope := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeReadUser)

	t.Run("Watch", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(tokenWithRepoScope)
		MakeRequest(t, req, http.StatusOK)
	})

	t.Run("GetWatchedRepos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/users/%s/subscriptions", user)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "1", resp.Header().Get("X-Total-Count"))

		var repos []api.Repository
		DecodeJSON(t, resp, &repos)
		assert.Len(t, repos, 1)
		assert.Equal(t, repo, repos[0].FullName)
	})

	t.Run("GetMyWatchedRepos", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", "/api/v1/user/subscriptions").
			AddTokenAuth(tokenWithRepoScope)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "1", resp.Header().Get("X-Total-Count"))

		var repos []api.Repository
		DecodeJSON(t, resp, &repos)
		assert.Len(t, repos, 1)
		assert.Equal(t, repo, repos[0].FullName)
	})

	t.Run("IsWatching", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/subscription", repo))
		MakeRequest(t, req, http.StatusUnauthorized)

		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(tokenWithRepoScope)
		MakeRequest(t, req, http.StatusOK)

		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/subscription", repo+"notexisting")).
			AddTokenAuth(tokenWithRepoScope)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("Unwatch", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(tokenWithRepoScope)
		MakeRequest(t, req, http.StatusNoContent)
	})
}

func TestAPIWatchEvents(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := "user1"
	repo := "user2/repo1"

	session := loginUser(t, user)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	t.Run("WatchWithDefaultEvents", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Watch without specifying events (should use defaults)
		req := NewRequest(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var watchInfo api.WatchInfo
		DecodeJSON(t, resp, &watchInfo)
		assert.True(t, watchInfo.Subscribed)
		// Default should be all events (7)
		assert.Equal(t, int64(repo_model.WatchEventAll), watchInfo.WatchEvents)

		// Clean up
		req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})

	t.Run("WatchWithSpecificEvents", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Watch with only issues events
		issuesOnly := int64(repo_model.WatchEventIssues)
		req := NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo), api.WatchOptions{
			WatchEvents: &issuesOnly,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var watchInfo api.WatchInfo
		DecodeJSON(t, resp, &watchInfo)
		assert.True(t, watchInfo.Subscribed)
		assert.Equal(t, int64(repo_model.WatchEventIssues), watchInfo.WatchEvents)

		// Clean up
		req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})

	t.Run("WatchWithMultipleEvents", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Watch with issues and PRs (bitmask: 1 + 2 = 3)
		issuesAndPRs := int64(repo_model.WatchEventIssues | repo_model.WatchEventPullRequests)
		req := NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo), api.WatchOptions{
			WatchEvents: &issuesAndPRs,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var watchInfo api.WatchInfo
		DecodeJSON(t, resp, &watchInfo)
		assert.True(t, watchInfo.Subscribed)
		assert.Equal(t, int64(3), watchInfo.WatchEvents)

		// Verify IsWatching returns the events
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)

		DecodeJSON(t, resp, &watchInfo)
		assert.Equal(t, int64(3), watchInfo.WatchEvents)

		// Clean up
		req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})

	t.Run("WatchReleasesOnly", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		// Watch with only releases events
		releasesOnly := int64(repo_model.WatchEventReleases)
		req := NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/repos/%s/subscription", repo), api.WatchOptions{
			WatchEvents: &releasesOnly,
		}).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var watchInfo api.WatchInfo
		DecodeJSON(t, resp, &watchInfo)
		assert.True(t, watchInfo.Subscribed)
		assert.Equal(t, int64(repo_model.WatchEventReleases), watchInfo.WatchEvents)

		// Clean up
		req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/repos/%s/subscription", repo)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNoContent)
	})
}
