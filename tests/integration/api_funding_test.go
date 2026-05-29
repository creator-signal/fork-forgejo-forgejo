// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func createFundingConfig(t *testing.T, user *user_model.User, repo *repo_model.Repository, treePath string, fundingConfig map[string]any) {
	t.Helper()

	config, err := yaml.Marshal(fundingConfig)
	require.NoError(t, err)

	err = createOrReplaceFileInBranch(user, repo, treePath, repo.DefaultBranch, string(config))
	require.NoError(t, err)
}

func getRepoFundingConfig(t *testing.T, repo *repo_model.Repository, token string) []*api.RepoFundingEntry {
	t.Helper()

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding", repo.OwnerName, repo.Name)

	req := NewRequest(t, "GET", urlStr).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var funding []*api.RepoFundingEntry

	DecodeJSON(t, resp, &funding)

	return funding
}

var testFundingCandidates = []string {
	".forgejo/FUNDING.yaml",
	".github/FUNDING.yaml",
	"FUNDING.yaml",

	".forgejo/FUNDING.yml",
	".github/FUNDING.yml",
	"FUNDING.yml",

	".forgejo/fUnDiNg.yaml",
	".github/fUnDiNg.yaml",
	"fUnDiNg.yaml",
}

// TODO: Test API responses when funding config is invalid
// TODO: Test API responses when there's both a valid and invalid funding config (the first one found should apply, regardless of whether it even has valid entries!)
// TODO: Test API responses when the config contains HTML-malicious entries (think XSS); the output must be valid URL matter! (our frontend interpolator already escapes the data, we should do the same for outgoing API responses too)
// TODO: test a provider limit of 0 (provider is disabled, never shows in API except in /settings/funding)
// TODO: Ensure providers are unique (duplicate "custom" entries get elided)

func TestAPIFundingSettings(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		t.Run("Global funding config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/settings/funding")
			resp := MakeRequest(t, req, http.StatusOK)

			var providers api.FundingSettings
			DecodeJSON(t, resp, &providers)

			assert.Len(t, providers.Providers, 3) // we have 3 default providers (smoke test to see that these decode correctly)
			// TODO: assert order is consistent too
		})
	})
}

func TestAPIRepoFunding(t *testing.T) {
	for _, treePath := range testFundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
			session := loginUser(t, owner.Name)
			token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

			assert.Empty(t, getRepoFundingConfig(t, repo, token))

			t.Run("Unknown repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/api/v1/repos/not/here/funding").AddTokenAuth(token)
				_ = MakeRequest(t, req, http.StatusNotFound)
			})

			// TODO: Private repo should also return 404

			t.Run("Empty", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				funding := getRepoFundingConfig(t, repo, token)

				assert.Empty(t, funding)
			})

			t.Run("Simple config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://example.com", funding[0].Text)
				assert.Equal(t, "https://example.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "ko_fi", funding[1].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[1].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[1].Icon)
			})

			t.Run("Custom string array", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				testSlice := make([]string, 2)
				testSlice[0] = "https://a.com"
				testSlice[1] = "https://b.com"

				config := make(map[string]any)
				config["custom"] = testSlice

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://b.com", funding[1].Text)
				assert.Equal(t, "https://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
			})

			t.Run("Invalid config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				testSlice := make([][]string, 1)
				testSlice[0] = []string{"test"}

				config := make(map[string]any)
				config["custom"] = testSlice

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Empty(t, funding)
			})

			t.Run("Partially invalid (bad key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["liberapay"] = "test"
				config["ko_fi"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 3)

				// no ko_fi, it's not a string value

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)

				assert.Equal(t, "liberapay", funding[2].ProviderName)
				assert.Equal(t, "liberapay.com/test", funding[2].Text)
				assert.Equal(t, "https://liberapay.com/test", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/liberapay.svg", funding[2].Icon)
			})

			t.Run("Partially invalid (unknown key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = "test"
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
			})

			t.Run("Partially invalid (bad and unknown key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
			})

			t.Run("Partially invalid (bad and unknown keys omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = "test"
				config["ko_fi"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				// no ko_fi, it's not a string
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
			})

			t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []any{42, "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 1)

				// no 42, it's not a string
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://example.com", funding[0].Text)
				assert.Equal(t, "https://example.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 4)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
			})

			t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "test"
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
			})

			t.Run("Partially invalid (too many of one provider, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test"}
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test", "test2"}
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// no custom/too_many or ko_fi/test2, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
			})
		})
	}
}

func TestAPIRepoValidateFunding(t *testing.T) {
	for _, treePath := range testFundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
			session := loginUser(t, owner.Name)
			token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

			t.Run("Unknown repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", "/api/v1/repos/not/here/funding/validate").AddTokenAuth(token)
				_ = MakeRequest(t, req, http.StatusNotFound)
			})

			// TODO: Private repo should also return 404

			urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", owner.Name, repo.Name)

			t.Run("Empty", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Valid (single key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Invalid (single key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				testSlice := make([][]string, 1)
				testSlice[0] = []string{"test"}

				config := make(map[string]any)
				config["custom"] = testSlice

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Invalid type for key 'custom', expected a string or string array")
				// TODO: feels weird sending API response data like this in only english.. send a list of issue code strings instead (maybe just our locale strings?), and document them enough for an API consumer to explain them to their users in their users' language.
			})

			t.Run("Partially invalid (single bad key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Invalid type for key 'ko_fi', expected a string or string array")
			})

			t.Run("Partially invalid (single unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = "test"
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Unknown funding provider: whatever")
			})

			t.Run("Partially invalid (single bad unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Unknown funding provider: whatever")
			})

			t.Run("Partially invalid (one bad and one unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["whatever"] = "test"
				config["ko_fi"] = 42
				config["custom"] = []string{"test", "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Invalid type for key 'ko_fi', expected a string or string array\nUnknown funding provider: whatever")
			})

			t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []any{42, "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Invalid type for key 'custom', expected a string or string array")
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Expected up to 4 of funding provider custom")
			})

			t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "test"
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Expected up to 4 of funding provider custom")
			})

			t.Run("Partially invalid (too many of one provider, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test"}
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Expected up to 4 of funding provider custom")
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test", "test2"}
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, fundingValidation.Message, "Expected up to 4 of funding provider custom\nExpected up to 1 of funding provider ko_fi")
			})
		})
	}
}
