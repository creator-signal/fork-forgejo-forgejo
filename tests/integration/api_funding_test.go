// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createFundingConfig(t *testing.T, user *user_model.User, repo *repo_model.Repository, treePath, config string) {
	t.Helper()

	err := createOrReplaceFileInBranch(user, repo, treePath, repo.DefaultBranch, config)
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

var testFundingCandidates = []string{
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

			assert.Len(t, providers.Providers, 12) // we have 12 default providers (smoke test to see that these decode correctly)

			names := make([]string, 0, len(providers.Providers))
			for _, provider := range providers.Providers {
				names = append(names, provider.Name)
			}
			assert.True(t, slices.IsSorted(names), "configured providers should be listed alphabetically")

			customIdx := slices.IndexFunc(providers.Providers, func(p *api.FundingProvider) bool {
				return p.Name == "custom"
			})
			custom := providers.Providers[customIdx]
			assert.NotNil(t, custom)
			assert.Equal(t, "/assets/img/svg/octicon-link.svg", custom.Icon)
			assert.Empty(t, custom.IconDark)
			assert.Equal(t, "%[1]s", custom.Text)
			assert.Equal(t, "%[1]s", custom.URL)
			assert.Equal(t, 4, int(custom.Limit))
			assert.Equal(t, ".+", custom.InputPattern)

			liberapayIdx := slices.IndexFunc(providers.Providers, func(p *api.FundingProvider) bool {
				return p.Name == "liberapay"
			})
			liberapay := providers.Providers[liberapayIdx]
			assert.NotNil(t, liberapay)
			assert.Equal(t, "/assets/img/funding/liberapay.svg", liberapay.Icon)
			assert.Empty(t, liberapay.IconDark)
			assert.Equal(t, "liberapay.com/%[1]s", liberapay.Text)
			assert.Equal(t, "https://liberapay.com/%[1]s", liberapay.URL)
			assert.Equal(t, 1, int(liberapay.Limit))
			assert.Equal(t, "^[^/]+$", liberapay.InputPattern)

			thanksDevIdx := slices.IndexFunc(providers.Providers, func(p *api.FundingProvider) bool {
				return p.Name == "thanks_dev"
			})
			thanksDev := providers.Providers[thanksDevIdx]
			assert.NotNil(t, thanksDev)
			assert.Equal(t, "/assets/img/funding/thanks_dev.svg", thanksDev.Icon)
			assert.Equal(t, "/assets/img/funding/thanks_dev_dark.svg", thanksDev.IconDark)
			assert.Equal(t, "thanks.dev/%[1]s", thanksDev.Text)
			assert.Equal(t, "https://thanks.dev/%[1]s", thanksDev.URL)
			assert.Equal(t, 1, int(thanksDev.Limit))
			assert.Equal(t, `^[^/]+\/[^/]+\/[^/]+$`, thanksDev.InputPattern)
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

			t.Run("Private repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile(`
ko_fi: example
liberapay: example
custom: example.com
`)
				repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
					Files:     mfs,
					IsPrivate: true,
				})

				req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/funding", repo.OwnerName, repo.Name))
				_ = MakeRequest(t, req, http.StatusNotFound)
			})

			t.Run("Empty repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				funding := getRepoFundingConfig(t, repo, token)

				assert.Empty(t, funding)
			})

			t.Run("Empty funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				createFundingConfig(t, owner, repo, treePath, ``)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Empty(t, funding)
			})

			t.Run("Simple config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: "https://example.com"` + "\n" +
					"patreon: test\n" +
					"ko_fi: test\n"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 3)

				// order is the same as given in the config
				custom := funding[0]
				patreon := funding[1]
				koFi := funding[2]

				assert.Equal(t, "custom", custom.ProviderName)
				assert.Equal(t, "https://example.com", custom.Text)
				assert.Equal(t, "https://example.com", custom.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", custom.Icon)
				assert.Empty(t, custom.IconDark)

				assert.Equal(t, "ko_fi", koFi.ProviderName)
				assert.Equal(t, "ko-fi.com/test", koFi.Text)
				assert.Equal(t, "https://ko-fi.com/test", koFi.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", koFi.Icon)
				assert.Empty(t, koFi.IconDark)

				assert.Equal(t, "patreon", patreon.ProviderName)
				assert.Equal(t, "patreon.com/test", patreon.Text)
				assert.Equal(t, "https://patreon.com/test", patreon.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon.svg", patreon.Icon)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon_dark.svg", patreon.IconDark) // patreon includes a dark-theme icon, whereas ko-fi does not
			})

			t.Run("Custom string array", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom:\n" +
					`- "https://a.com"` + "\n" +
					"- b.com\n" +
					`- "http://withquery.example.com?test=foo"` + "\n" +
					`- "http://thistimewithhash#foo"` + "\n"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 4)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "b.com", funding[1].Text)
				assert.Equal(t, "http://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "http://withquery.example.com?test=foo", funding[2].Text)
				assert.Equal(t, "http://withquery.example.com?test=foo", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Empty(t, funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "http://thistimewithhash#foo", funding[3].Text)
				assert.Equal(t, "http://thistimewithhash#foo", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Empty(t, funding[3].IconDark)
			})

			t.Run("Skips duplicate entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: ["https://a.com", "https://a.com", "https://b.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://a.com", funding[0].Text)
				assert.Equal(t, "https://a.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://b.com", funding[1].Text)
				assert.Equal(t, "https://b.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)
			})

			t.Run("Invalid config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [[test]]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Empty(t, funding)
			})

			t.Run("Partially invalid (bad key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "liberapay: test\n" +
					"ko_fi: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 3)

				// no ko_fi, it's not a string value

				liberapay := funding[0]
				custom1 := funding[1]
				custom2 := funding[2]

				assert.Equal(t, "custom", custom1.ProviderName)
				assert.Equal(t, "test", custom1.Text)
				assert.Equal(t, "http://test", custom1.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", custom1.Icon)
				assert.Empty(t, custom1.IconDark)

				assert.Equal(t, "custom", custom2.ProviderName)
				assert.Equal(t, "https://example.com", custom2.Text)
				assert.Equal(t, "https://example.com", custom2.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", custom2.Icon)
				assert.Empty(t, custom2.IconDark)

				assert.Equal(t, "liberapay", liberapay.ProviderName)
				assert.Equal(t, "liberapay.com/test", liberapay.Text)
				assert.Equal(t, "https://liberapay.com/test", liberapay.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/liberapay.svg", liberapay.Icon)
				assert.Empty(t, liberapay.IconDark)
			})

			t.Run("Partially invalid (unknown key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: test\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)
			})

			t.Run("Partially invalid (bad and unknown key omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)
			})

			t.Run("Partially invalid (bad and unknown keys omitted)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: test\n" +
					"ko_fi: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 2)

				// no whatever, it's not a known value
				// no ko_fi, it's not a string
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test", funding[0].Text)
				assert.Equal(t, "http://test", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)
			})

			t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [42, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 1)

				// no 42, it's not a string
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "https://example.com", funding[0].Text)
				assert.Equal(t, "https://example.com", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 4)

				// no too_many, we have enough
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "test1", funding[0].Text)
				assert.Equal(t, "http://test1", funding[0].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com", funding[1].Text)
				assert.Equal(t, "https://example.com", funding[1].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "test3", funding[2].Text)
				assert.Equal(t, "http://test3", funding[2].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Empty(t, funding[2].IconDark)

				assert.Equal(t, "custom", funding[3].ProviderName)
				assert.Equal(t, "test4", funding[3].Text)
				assert.Equal(t, "http://test4", funding[3].URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[3].Icon)
				assert.Empty(t, funding[3].IconDark)
			})

			t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: test\n" +
					"patreon: test\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 6)

				koFi := funding[0]
				patreon := funding[1]
				test1 := funding[2]
				exampleCom := funding[3]
				test3 := funding[4]
				test4 := funding[5]
				// no too_many, we have enough from "custom"

				assert.Equal(t, "custom", test1.ProviderName)
				assert.Equal(t, "test1", test1.Text)
				assert.Equal(t, "http://test1", test1.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test1.Icon)
				assert.Empty(t, test1.IconDark)

				assert.Equal(t, "custom", exampleCom.ProviderName)
				assert.Equal(t, "https://example.com", exampleCom.Text)
				assert.Equal(t, "https://example.com", exampleCom.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", exampleCom.Icon)
				assert.Empty(t, exampleCom.IconDark)

				assert.Equal(t, "custom", test3.ProviderName)
				assert.Equal(t, "test3", test3.Text)
				assert.Equal(t, "http://test3", test3.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test3.Icon)
				assert.Empty(t, test3.IconDark)

				assert.Equal(t, "custom", test4.ProviderName)
				assert.Equal(t, "test4", test4.Text)
				assert.Equal(t, "http://test4", test4.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test4.Icon)
				assert.Empty(t, test4.IconDark)

				assert.Equal(t, "ko_fi", koFi.ProviderName)
				assert.Equal(t, "ko-fi.com/test", koFi.Text)
				assert.Equal(t, "https://ko-fi.com/test", koFi.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", koFi.Icon)
				assert.Empty(t, koFi.IconDark)

				assert.Equal(t, "patreon", patreon.ProviderName)
				assert.Equal(t, "patreon.com/test", patreon.Text)
				assert.Equal(t, "https://patreon.com/test", patreon.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon.svg", patreon.Icon)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/patreon_dark.svg", patreon.IconDark)
			})

			t.Run("Partially invalid (too many of two providers, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: [test]\n" +
					"tidelift: npm/example\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				koFi := funding[0]
				test1 := funding[1]
				exampleCom := funding[2]
				test3 := funding[3]
				test4 := funding[4]

				assert.Equal(t, "custom", test1.ProviderName)
				assert.Equal(t, "test1", test1.Text)
				assert.Equal(t, "http://test1", test1.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test1.Icon)
				assert.Empty(t, test1.IconDark)

				assert.Equal(t, "custom", exampleCom.ProviderName)
				assert.Equal(t, "https://example.com", exampleCom.Text)
				assert.Equal(t, "https://example.com", exampleCom.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", exampleCom.Icon)
				assert.Empty(t, exampleCom.IconDark)

				assert.Equal(t, "custom", test3.ProviderName)
				assert.Equal(t, "test3", test3.Text)
				assert.Equal(t, "http://test3", test3.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test3.Icon)
				assert.Empty(t, test3.IconDark)

				assert.Equal(t, "custom", test4.ProviderName)
				assert.Equal(t, "test4", test4.Text)
				assert.Equal(t, "http://test4", test4.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test4.Icon)
				assert.Empty(t, test4.IconDark)

				assert.Equal(t, "ko_fi", koFi.ProviderName)
				assert.Equal(t, "ko-fi.com/test", koFi.Text)
				assert.Equal(t, "https://ko-fi.com/test", koFi.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", koFi.Icon)
				assert.Empty(t, koFi.IconDark)
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: [test, test2]\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 5)

				koFi := funding[0]
				test1 := funding[1]
				exampleCom := funding[2]
				test3 := funding[3]
				test4 := funding[4]
				// no too_many or ko_fi/test2, we have enough

				assert.Equal(t, "custom", test1.ProviderName)
				assert.Equal(t, "test1", test1.Text)
				assert.Equal(t, "http://test1", test1.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test1.Icon)
				assert.Empty(t, test1.IconDark)

				assert.Equal(t, "custom", exampleCom.ProviderName)
				assert.Equal(t, "https://example.com", exampleCom.Text)
				assert.Equal(t, "https://example.com", exampleCom.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", exampleCom.Icon)
				assert.Empty(t, exampleCom.IconDark)

				assert.Equal(t, "custom", test3.ProviderName)
				assert.Equal(t, "test3", test3.Text)
				assert.Equal(t, "http://test3", test3.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test3.Icon)
				assert.Empty(t, test3.IconDark)

				assert.Equal(t, "custom", test4.ProviderName)
				assert.Equal(t, "test4", test4.Text)
				assert.Equal(t, "http://test4", test4.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", test4.Icon)
				assert.Empty(t, test4.IconDark)

				assert.Equal(t, "ko_fi", koFi.ProviderName)
				assert.Equal(t, "ko-fi.com/test", koFi.Text)
				assert.Equal(t, "https://ko-fi.com/test", koFi.URL)
				assert.Equal(t, setting.AppSubURL+"/assets/img/funding/ko_fi.svg", koFi.Icon)
				assert.Empty(t, koFi.IconDark)
			})

			t.Run("Bad URLs get escaped or elided", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `ko_fi: '"><script>alert(1);</script><a class="'` + "\n" + // omitted (contains a `/`)
					"liberapay: 'text/other'\n" + // omitted (contains a `/`)
					"thanks_dev: 'could/be/real/bad'\n" + // omitted (too many `/`)
					"custom:\n" +
					`- '#" style="background: url(localhost)'` + "\n" +
					`- 'https://example.com" class="rogue injection'` + "\n" + // omitted (space in domain name)
					`- 'https://example.com/" class="rogue injection'` + "\n" + // URL escaped
					"- \"<script>alert`1`</script>\""
				createFundingConfig(t, owner, repo, treePath, config)

				funding := getRepoFundingConfig(t, repo, token)
				assert.Len(t, funding, 3)

				// omits values that don't parse as URLs.
				// returned URL values are always valid, but it's the API consumer's responsibility to escape Text for its presentation context (e.g. HTML)
				assert.Equal(t, "custom", funding[0].ProviderName)
				assert.Equal(t, "http://#%22%20style=%22background:%20url(localhost)", funding[0].URL)
				assert.Equal(t, "#\" style=\"background: url(localhost)", funding[0].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[0].Icon)
				assert.Empty(t, funding[0].IconDark)

				assert.Equal(t, "custom", funding[1].ProviderName)
				assert.Equal(t, "https://example.com/%22%20class=%22rogue%20injection", funding[1].URL)
				assert.Equal(t, "https://example.com/\" class=\"rogue injection", funding[1].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[1].Icon)
				assert.Empty(t, funding[1].IconDark)

				assert.Equal(t, "custom", funding[2].ProviderName)
				assert.Equal(t, "http://%3Cscript%3Ealert%601%60%3C/script%3E", funding[2].URL)
				assert.Equal(t, "<script>alert`1`</script>", funding[2].Text)
				assert.Equal(t, setting.AppSubURL+"/assets/img/svg/octicon-link.svg", funding[2].Icon)
				assert.Empty(t, funding[2].IconDark)
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

			t.Run("Private repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile(`
ko_fi: example
liberapay: example
custom: example.com
`)
				repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
					Files:     mfs,
					IsPrivate: true,
				})

				req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", repo.OwnerName, repo.Name))
				_ = MakeRequest(t, req, http.StatusNotFound)
			})

			urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", owner.Name, repo.Name)

			t.Run("Empty repo", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Empty funding config", func(t *testing.T) {
				createFundingConfig(t, owner, repo, treePath, ``)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Valid (single key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: "https://example.com"`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.True(t, fundingValidation.Valid)
				assert.Empty(t, fundingValidation.Message)
			})

			t.Run("Invalid (single key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [["test"]]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", fundingValidation.Message)
			})

			t.Run("Partially invalid (single bad key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", fundingValidation.Message)
			})

			t.Run("Partially invalid (single unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: test\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Unknown funding provider: whatever", fundingValidation.Message)
			})

			t.Run("Partially invalid (single bad unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Unknown funding provider: whatever", fundingValidation.Message)
			})

			t.Run("Partially invalid (one bad and one unknown key)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "whatever: test\n" +
					"ko_fi: 42\n" +
					`custom: [test, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Unknown funding provider: whatever\nInvalid type for key 'ko_fi', expected a string or string array", fundingValidation.Message)
			})

			t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [42, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of two providers, valid others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: test\n" +
					"tidelift: 'npm/example'\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Funding provider tidelift is not allowed\nExpected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of one provider, valid list of others)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: [test]\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: [test, test2]\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Expected up to 1 of funding provider ko_fi\nExpected up to 4 of funding provider custom", fundingValidation.Message)
			})

			t.Run("Partially invalid (duplicate entries)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: ["https://a.com", "https://a.com", "https://b.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Duplicate entry for key 'custom': https://a.com", fundingValidation.Message)
			})

			t.Run("Bad URLs are may cause invalid config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `ko_fi: '"><script>alert(1);</script><a class="'` + "\n" + // omitted (contains a `/`)
					"liberapay: 'text/other'\n" + // omitted (contains a `/`)
					"thanks_dev: 'could/be/real/bad'\n" + // omitted (too many `/`)
					"custom:\n" +
					`- '#" style="background: url(localhost)'` + "\n" +
					`- 'https://example.com" class="rogue injection'` + "\n" + // omitted (space in domain name)
					`- 'https://example.com/" class="rogue injection'` + "\n" + // URL escaped
					"- \"<script>alert`1`</script>\""
				createFundingConfig(t, owner, repo, treePath, config)

				resp := MakeRequest(t, NewRequest(t, "GET", urlStr).AddTokenAuth(token), http.StatusOK)

				var fundingValidation api.ConfigValidation
				DecodeJSON(t, resp, &fundingValidation)

				assert.False(t, fundingValidation.Valid)
				assert.Equal(t, "Value for key 'ko_fi' does not match pattern /^[^/]+$/\n"+
					"Value for key 'liberapay' does not match pattern /^[^/]+$/\n"+
					`Value for key 'thanks_dev' does not match pattern /^[^/]+\/[^/]+\/[^/]+$/`+"\n"+
					`Invalid URL value for key 'custom': parse "https://example.com\" class=\"rogue injection": invalid character " " in host name`,
					fundingValidation.Message)
			})
		})
	}
}
