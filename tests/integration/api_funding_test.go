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

func TestAPIFundingSettings(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		t.Run("Global funding config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/settings/funding")
			resp := MakeRequest(t, req, http.StatusOK)

			var providers api.FundingSettings
			DecodeJSON(t, resp, &providers)

			assert.Len(t, providers.Providers, 10) // we have 10 default providers (smoke test to see that these decode correctly)
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
				config["patreon"] = "test"
				config["ko_fi"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 3)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://example.com", funding[0].Text)
				assert.Equal(t, "https://example.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "ko_fi", funding[1].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[1].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "patreon", funding[2].ProviderName)
				assert.Equal(t, "patreon.com/test", funding[2].Text)
				assert.Equal(t, "https://patreon.com/test", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon.svg", funding[2].Icon)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon_dark.svg", funding[2].IconDark) // patreon includes a dark-theme icon, whereas ko-fi does not
			})

			t.Run("Custom string array", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{
					"https://a.com",
					"b.com",
					"http://withquery.example.com?test=foo",
					"http://thistimewithhash#foo",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 4)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "b.com", funding[1].Text)
				assert.Equal(t, "http://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "http://withquery.example.com?test=foo", funding[2].Text)
				assert.Equal(t, "http://withquery.example.com?test=foo", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "http://thistimewithhash#foo", funding[3].Text)
				assert.Equal(t, "http://thistimewithhash#foo", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)
			})

			t.Run("Skips duplicate entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"https://a.com", "https://a.com", "https://b.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://b.com", funding[1].Text)
				assert.Equal(t, "https://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)
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
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "liberapay", funding[2].ProviderName)
				assert.Equal(t, "liberapay.com/test", funding[2].Text)
				assert.Equal(t, "https://liberapay.com/test", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/liberapay.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)
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
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)
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
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)
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
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)
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
				assert.Equal(t, "", funding[0].IconDark)
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 4)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "http://test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "http://test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "http://test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)
			})

			t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "test"
				config["patreon"] = "test"
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 6)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "http://test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "http://test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "http://test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
				assert.Equal(t, "", funding[4].IconDark)

				assert.Equal(t, "patreon", funding[5].ProviderName)
				assert.Equal(t, "patreon.com/test", funding[5].Text)
				assert.Equal(t, "https://patreon.com/test", funding[5].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon.svg", funding[5].Icon)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon_dark.svg", funding[5].IconDark)
			})

			t.Run("Partially invalid (too many of one provider, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test"}
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "http://test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "http://test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "http://test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
				assert.Equal(t, "", funding[4].IconDark)
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test", "test2"}
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// no custom/too_many or ko_fi/test2, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "http://test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "http://test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "http://test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)

				assert.Equal(t, "ko_fi", funding[4].ProviderName)
				assert.Equal(t, "ko-fi.com/test", funding[4].Text)
				assert.Equal(t, "https://ko-fi.com/test", funding[4].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[4].Icon)
				assert.Equal(t, "", funding[4].IconDark)
			})

			t.Run("Bad URLs get escaped or elided", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "\"><script>alert(1);</script><a class=\"" // URL escaped
				config["liberapay"] = "text/other" // URL escaped // TODO: Should this maybe just do without instead? When do we need to support multiple path segments here anyway? Tidelift? Should we permit arbitrary paths, or should we specifically allow these for particular providers?
				config["custom"] = []string{
					"#\" style=\"background: url(localhost)",
					"https://example.com\" class=\"rogue injection", // omitted (space in domain name)
					"https://example.com/\" class=\"rogue injection", // URL escaped
					"<script>alert`1`</script>",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				// omits values that don't parse as URLs.
				// returned URL values are always valid, but it's the API consumer's responsibility to escape Text for its presentation context (e.g. HTML)
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "http://#%22%20style=%22background:%20url(localhost)", funding[0].URL)
				assert.Equal(t, "#\" style=\"background: url(localhost)", funding[0].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Equal(t, "", funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com/%22%20class=%22rogue%20injection", funding[1].URL)
				assert.Equal(t, "https://example.com/\" class=\"rogue injection", funding[1].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Equal(t, "", funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "http://%3Cscript%3Ealert%601%60%3C/script%3E", funding[2].URL)
				assert.Equal(t, "<script>alert`1`</script>", funding[2].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Equal(t, "", funding[2].IconDark)

				assert.Equal(t, "ko_fi", funding[3].ProviderName)
				assert.Equal(t, "https://ko-fi.com/%22%3E%3Cscript%3Ealert%281%29%3B%3C%2Fscript%3E%3Ca%20class=%22", funding[3].URL)
				assert.Equal(t, "ko-fi.com/\"><script>alert(1);</script><a class=\"", funding[3].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", funding[3].Icon)
				assert.Equal(t, "", funding[3].IconDark)

				assert.Equal(t, "liberapay", funding[4].ProviderName)
				assert.Equal(t, "https://liberapay.com/text%2Fother", funding[4].URL)
				assert.Equal(t, "liberapay.com/text/other", funding[4].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/liberapay.svg", funding[4].Icon)
				assert.Equal(t, "", funding[4].IconDark)
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
				assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", fundingValidation.Message)
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
				assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", fundingValidation.Message)
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
				assert.Equal(t, "Unknown funding provider: whatever", fundingValidation.Message)
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
				assert.Equal(t, "Unknown funding provider: whatever", fundingValidation.Message)
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
				assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array\nUnknown funding provider: whatever", fundingValidation.Message)
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
				assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "test"
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of one provider, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test"}
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test", "test2"}
				config["custom"] = []string{
					"test1",
					"https://example.com",
					"test3",
					"test4",
					"too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom\nExpected up to 1 of funding provider ko_fi", fundingValidation.Message)
			})

			t.Run("Partially invalid (duplicate entries)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"https://a.com", "https://a.com", "https://b.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Duplicate entry for key 'custom': https://a.com", fundingValidation.Message)
			})

			t.Run("Bad URLs are may cause invalid config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "\"><script>alert(1);</script><a class=\""
				config["liberapay"] = "text/other"
				config["custom"] = []string{
					"#\" style=\"background: url(localhost)",
					"https://example.com\" class=\"rogue injection",
					"https://example.com/\" class=\"rogue injection",
					"<script>alert`1`</script>",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, `Invalid URL value for key 'custom': parse "https://example.com\" class=\"rogue injection": invalid character " " in host name`, fundingValidation.Message)
			})
		})
	}
}
