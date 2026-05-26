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
	resp := MakeRequest(t, req, http.StatusOK) // FIXME: Status is NOT always 200!

	var funding []*api.RepoFundingEntry

	DecodeJSON(t, resp, &funding)

	return funding
}

var fundingCandidates = []string {
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
// TODO: Is a config invalid if it contains additional keys? (yes, but we still get the rest so it's fine)
// TODO: Is a config invalid if it contains additional keys with invalid values? (yes, but we still get the rest so it's fine) (also, we don't care about the type of an invalid key, it's already invalid)
// TODO: Test API responses when there's both a valid and invalid funding config
// TODO: Test API responses when one repo has a funding config but the target does not
// TODO: Test API responses when one repo (the target) has a funding config, but another does not
// TODO: Test API responses when the config contains entries for unknown providers
// TODO: Test API responses when the config contains HTML-malicious entries (think XSS); the output must be valid URL matter!
// TODO: Test API 404 for unknown repo
// TODO: Test that funding entries are in the same order as they were defined in the config
// TODO: Allow only up to 4 Custom entries in a config
// TODO: Allow only one entry for all keys except Custom

func TestAPIRepoFunding(t *testing.T) {
	for _, treePath := range fundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
			session := loginUser(t, owner.Name)
			token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

			assert.Empty(t, getRepoFundingConfig(t, repo, token))

			t.Run("Empty", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				funding := getRepoFundingConfig(t, repo, token)

				assert.Empty(t, funding)
			})

			t.Run("SimpleConfig", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "https://example.com", funding[0].Text)
				assert.Equal(t, "https://example.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "Ko-Fi/test", funding[1].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[1].Icon)
			})

			t.Run("StringArray", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				testSlice := make([]string, 2)
				testSlice[0] = "https://a.com"
				testSlice[1] = "https://b.com"

				config := make(map[string]any)
				config["custom"] = testSlice

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)

				assert.Equal(t, "https://b.com", funding[1].Text)
				assert.Equal(t, "https://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
			})
		})
	}
}

func TestAPIRepoValidateFunding(t *testing.T) {
	for _, treePath := range fundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
			owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
			session := loginUser(t, owner.Name)
			token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

			urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", owner.Name, repo.Name)

			t.Run("Empty", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Valid", func(t *testing.T) {
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

			t.Run("Invalid", func(t *testing.T) {
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
				assert.NotEmpty(t, fundingValidation.Message)
			})
		})
	}
}
