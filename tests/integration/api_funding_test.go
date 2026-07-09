// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/url"
	"slices"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getRepoFundingConfig(t *testing.T, repo *repo_model.Repository, token string) []*api.RepoFundingEntry {
	t.Helper()

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding", repo.OwnerName, repo.Name)

	req := NewRequest(t, "GET", urlStr).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var funding []*api.RepoFundingEntry
	DecodeJSON(t, resp, &funding)

	return funding
}

func validateRepoFundingConfig(t *testing.T, repo *repo_model.Repository, token string) *api.ConfigValidation {
	t.Helper()

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", repo.OwnerName, repo.Name)

	req := NewRequest(t, "GET", urlStr).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusOK)

	var validation *api.ConfigValidation
	DecodeJSON(t, resp, &validation)

	return validation
}

func TestAPIFundingSettings(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		t.Run("Global funding config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/settings/funding")
			resp := MakeRequest(t, req, http.StatusOK)

			var providers api.FundingSettings
			DecodeJSON(t, resp, &providers)

			assert.Len(t, providers.Providers, 11) // we have 11 default providers (smoke test to see that these decode correctly)

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
			assert.Equal(t, "%[1]s", custom.Title)
			assert.Equal(t, "%[1]s", custom.Template)
			assert.Equal(t, ".+", custom.InputPattern)

			liberapayIdx := slices.IndexFunc(providers.Providers, func(p *api.FundingProvider) bool {
				return p.Name == "liberapay"
			})
			liberapay := providers.Providers[liberapayIdx]
			assert.NotNil(t, liberapay)
			assert.Equal(t, "liberapay.com/%[1]s", liberapay.Title)
			assert.Equal(t, "https://liberapay.com/%[1]s", liberapay.Template)
			assert.Equal(t, "^[^/]+$", liberapay.InputPattern)

			thanksDevIdx := slices.IndexFunc(providers.Providers, func(p *api.FundingProvider) bool {
				return p.Name == "thanks_dev"
			})
			thanksDev := providers.Providers[thanksDevIdx]
			assert.NotNil(t, thanksDev)
			assert.Equal(t, "thanks.dev/%[1]s", thanksDev.Title)
			assert.Equal(t, "https://thanks.dev/%[1]s", thanksDev.Template)
			assert.Equal(t, `^[^/]+\/[^/]+\/[^/]+$`, thanksDev.InputPattern)
		})
	})
}

var testFundingCandidates = []string{
	// a few tests to prove that the config is case-insensitive, then other tests can just pick one
	// a file extension gets appended to each of these, e.g. ".forgejo/FUNDING.yml"
	".forgejo/FUNDING",
	".forgejo/Funding",
	".github/FUNDING",
	".github/Funding",
	"FUNDING",
	"Funding",
}

func TestAPIRepoFundingConfigPaths(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		for _, candidate := range testFundingCandidates {
			for _, extension := range []string{".yaml", ".yml", ".YAML"} {
				treePath := candidate + extension
				t.Run(treePath, func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					// random config data proves we're reading the config using a new file, and that previous iterations aren't somehow polluting this test by mistake
					acctName := StringWithCharset(5+rand.IntN(10), "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

					config := fmt.Sprintf("ko_fi: %s", acctName)
					repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
						Files: forgery.MapFS{treePath: forgery.MapFile(config)},
					})

					fundingConfig := getRepoFundingConfig(t, repo, "")
					validation := validateRepoFundingConfig(t, repo, "")

					assert.True(t, validation.Valid)
					assert.Empty(t, validation.Message)

					assert.Len(t, fundingConfig, 1)
					koFi := fundingConfig[0]
					assert.Equal(t, "ko_fi", koFi.ProviderName)
					assert.Equal(t, fmt.Sprintf("ko-fi.com/%s", acctName), koFi.Title)
					assert.Equal(t, fmt.Sprintf("https://ko-fi.com/%s", acctName), koFi.Value)
				})
			}
		}
	})
}

func TestAPIRepoFundingGone(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		repo := forgery.CreateRepository(t, nil, nil)
		session := loginUser(t, repo.Owner.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

		// this repo has no funding config as of yet
		require.Empty(t, getRepoFundingConfig(t, repo, token))

		t.Run("Unknown repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/api/v1/repos/not/here/funding").AddTokenAuth(token)
			_ = MakeRequest(t, req, http.StatusNotFound)

			req = NewRequest(t, "GET", "/api/v1/repos/not/here/funding/validate").AddTokenAuth(token)
			_ = MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("Private repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			config := "ko_fi: example\n" +
				"liberapay: example\n" +
				"custom: example.com\n"
			repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
				Files:     forgery.MapFS{"FUNDING.yml": forgery.MapFile(config)},
				IsPrivate: true,
			})

			req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/funding", repo.OwnerName, repo.Name))
			_ = MakeRequest(t, req, http.StatusNotFound)

			req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/funding/validate", repo.OwnerName, repo.Name))
			_ = MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("Empty repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo := forgery.CreateRepository(t, nil, nil)
			funding := getRepoFundingConfig(t, repo, "")
			validation := validateRepoFundingConfig(t, repo, "")

			assert.True(t, validation.Valid)
			assert.Empty(t, validation.Message)
			assert.Empty(t, funding)
		})

		t.Run("Init repo", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
				Files: forgery.FilesInit{},
			})
			funding := getRepoFundingConfig(t, repo, "")
			validation := validateRepoFundingConfig(t, repo, "")

			assert.True(t, validation.Valid)
			assert.Empty(t, validation.Message)
			assert.Empty(t, funding)
		})

		t.Run("Empty funding config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
				Files: forgery.MapFS{"Funding.yml": forgery.MapFile("")},
			})
			funding := getRepoFundingConfig(t, repo, token)
			validation := validateRepoFundingConfig(t, repo, token)

			assert.True(t, validation.Valid)
			assert.Empty(t, validation.Message)
			assert.Empty(t, funding)
		})
	})
}

func assertEntry(t *testing.T, entry *api.RepoFundingEntry, expectedProvider, expectedTitle, expectedValue string) {
	t.Helper()
	assert.Equal(t, expectedProvider, entry.ProviderName)
	assert.Equal(t, expectedTitle, entry.Title)
	assert.Equal(t, expectedValue, entry.Value)
}

func assertCustom(t *testing.T, entry *api.RepoFundingEntry, expectedTitle, expectedValue string) {
	t.Helper()
	assertEntry(t, entry, "custom", expectedTitle, expectedValue)
}

func assertKoFi(t *testing.T, entry *api.RepoFundingEntry, expectedTitle, expectedValue string) {
	t.Helper()
	assertEntry(t, entry, "ko_fi", expectedTitle, expectedValue)
}

func assertPatreon(t *testing.T, entry *api.RepoFundingEntry, expectedTitle, expectedValue string) {
	t.Helper()
	assertEntry(t, entry, "patreon", expectedTitle, expectedValue)
}

func TestAPIRepoFundingConfigBasics(t *testing.T) {
	treePath := ".forgejo/Funding.yml"
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
			Files: forgery.FilesInit{},
		})
		owner := repo.Owner
		session := loginUser(t, owner.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

		// this repo has no funding config as of yet
		require.Empty(t, getRepoFundingConfig(t, repo, token))

		t.Run("Simple config", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			config := `custom: "https://example.com"` + "\n" +
				"patreon: test\n" +
				"ko_fi: test\n"
			err := createOrReplaceFileInBranch(owner, repo, treePath, repo.DefaultBranch, config)
			require.NoError(t, err)
			funding := getRepoFundingConfig(t, repo, token)
			validation := validateRepoFundingConfig(t, repo, token)

			assert.True(t, validation.Valid)
			assert.Empty(t, validation.Message)

			// order is the same as given in the config
			assert.Len(t, funding, 3)
			assertCustom(t, funding[0], "https://example.com", "https://example.com")
			assertPatreon(t, funding[1], "patreon.com/test", "https://patreon.com/test")
			assertKoFi(t, funding[2], "ko-fi.com/test", "https://ko-fi.com/test")
		})

		t.Run("Bad URLs get escaped or elided", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			config := `ko_fi: '"><script>alert(1);</script><a class="'` + "\n" + // omitted (contains a `/`)
				"liberapay: 'text/other'\n" + // omitted (contains a `/`)
				"thanks_dev: 'could/be/real/bad'\n" + // omitted (too many `/`)
				"custom:\n" +
				`- '#" style="background: url(localhost)'` + "\n" + // this is just a hash segment tbh
				`- 'https://example.com" class="rogue injection'` + "\n" + // omitted (space in domain name)
				`- 'https://example.com/" class="rogue injection'` + "\n" + // path gets URL escaped
				"- \"<script>alert`1`</script>\"\n" + // omitted ("`" in domain name)
				"- \"Arbitrary: text\"" // omitted (": text" is not a port number)
			err := createOrReplaceFileInBranch(owner, repo, treePath, repo.DefaultBranch, config)
			require.NoError(t, err)
			funding := getRepoFundingConfig(t, repo, token)
			validation := validateRepoFundingConfig(t, repo, token)

			// omits values that don't parse as URLs whose provider expects a URL.
			// returned URL values are always valid, but it's the API consumer's responsibility to escape Text for its presentation context (e.g. HTML)
			assert.False(t, validation.Valid)
			assert.Equal(t, "Value for key 'ko_fi' does not match pattern /^[^/]+$/\n"+
				"Value for key 'liberapay' does not match pattern /^[^/]+$/\n"+
				`Value for key 'thanks_dev' does not match pattern /^[^/]+\/[^/]+\/[^/]+$/`+"\n"+
				`Invalid URL value for key 'custom': parse "https://example.com\" class=\"rogue injection": invalid character " " in host name`+"\n"+
				`Invalid URL value for key 'custom': parse "http://<script>alert`+"`1`"+`</script>": invalid character "`+"`"+`" in host name`+"\n"+
				`Invalid URL value for key 'custom': parse "http://Arbitrary: text": invalid port ": text" after host`,
				validation.Message)

			assert.Len(t, funding, 2)
			assertCustom(t, funding[0], `#" style="background: url(localhost)`, "http:#%22%20style=%22background:%20url(localhost)")
			assertCustom(t, funding[1], `https://example.com/" class="rogue injection`, "https://example.com/%22%20class=%22rogue%20injection")
		})
	})
}
