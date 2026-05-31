// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"
	"forgejo.org/tests/forgery"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

// Returns the main HTML webpage for the given repo.
func getRepoPage(t *testing.T, repo *repo_model.Repository) *HTMLDoc {
	t.Helper()

	req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
	resp := MakeRequest(t, req, http.StatusOK)
	return NewHTMLParser(t, resp.Body)
}

// Returns the main HTML webpage for the given user profile.
func getUserPage(t *testing.T, user *user_model.User) *HTMLDoc {
	t.Helper()

	req := NewRequest(t, "GET", fmt.Sprintf("/%s", user.Name))
	resp := MakeRequest(t, req, http.StatusOK)
	return NewHTMLParser(t, resp.Body)
}

// Ensures the page has neither a Sponsor button nor a Sponsor modal.
func assertNoFunding(t *testing.T, htmlDoc *HTMLDoc) {
	htmlDoc.AssertElement(t, "button.sponsor", false)
	htmlDoc.AssertElement(t, "dialog#sponsor-modal", false)
}

// Ensures the page contains a Sponsor button.
func assertSponsorButton(t *testing.T, htmlDoc *HTMLDoc) *goquery.Selection {
	sponsorButton := htmlDoc.Find("button.sponsor")
	assert.Equal(t, 1, sponsorButton.Length())
	assert.Contains(t, sponsorButton.Text(), "Sponsor")
	htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

	return sponsorButton
}

// Ensures the page's Sponsor modal contains the given entry.
//
// `nth` is 1-indexed, indicating which entry in the list to check.
//
// If `imgSrc` is empty, then the entry is assumed to be a Custom URL entry,
// and an octicon-link SVG will be checked for instead of an image icon.
func assertFundingEntry(t *testing.T, htmlDoc *HTMLDoc, nth uint, href string, imgSrc string) {
	sel := fmt.Sprintf("dialog#sponsor-modal li:nth-child(%d)", nth)

	htmlDoc.AssertAttrEqual(t, sel + " a", "href", href)
	if imgSrc == "" {
		htmlDoc.AssertElement(t, sel + " img", false)
		htmlDoc.AssertElement(t, sel + " svg.octicon-link", true)
	} else {
		htmlDoc.AssertAttrEqual(t, sel + " img", "src", imgSrc)
		htmlDoc.AssertElement(t, sel + " svg.octicon-link", false)
	}
}

func TestSponsorButton(t *testing.T) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	t.Run("sponsor button hidden without funding config (repo)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			htmlDoc := getRepoPage(t, repo)
			assertNoFunding(t, htmlDoc)
		})
	})

	t.Run("sponsor button hidden without funding config (user)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			htmlDoc := getUserPage(t, owner)
			assertNoFunding(t, htmlDoc)
		})
	})

	t.Run("sponsor button shown with one valid and one invalid config", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			config := make(map[string]any)
			config["custom"] = "https://example.com"
			config["ko_fi"] = "test"
			createFundingConfig(t, owner, repo, ".forgejo/FUNDING.yml", config)

			config = make(map[string]any)
			config["custom"] = 42
			createFundingConfig(t, owner, repo, "FUNDING.yml", config)

			htmlDoc := getRepoPage(t, repo)
			assertSponsorButton(t, htmlDoc)

			// e2e tests check open/close behavior and accessibility, here we check data
			sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
			assert.Equal(t, 1, sponsorModalHeader.Length())
			assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

			sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
			assert.Equal(t, 2, sponsorEntries.Length())
			assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")
			assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")
		})
	})

	for _, treePath := range testFundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			t.Run("sponsor button shown on user profile", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				// first, without the special .profile repo:
				user := forgery.CreateUser(t, &forgery.CreateUserOptions{IsAdmin: false})
				htmlDoc := getUserPage(t, user)
				assertNoFunding(t, htmlDoc)

				// then, with the user's special .profile repo:
				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile(`
ko_fi: example
liberapay: example
custom: "https://example.com"
`)
				repo := forgery.CreateRepository(t, user, &forgery.CreateRepositoryOptions{
					Name: ".profile",
					Files: mfs,
				})

				htmlDoc = getUserPage(t, user)
				assertSponsorButton(t, htmlDoc)

				sponsorModal := htmlDoc.Find("dialog#sponsor-modal")
				assert.Equal(t, 1, sponsorModal.Length())

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Equal(t, fmt.Sprintf("Sponsor %s", user.Name), strings.TrimSpace(sponsorModalHeader.Text()))

				// includes the whole config
				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 3, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/example", "/assets/img/funding/ko_fi.svg")
				assertFundingEntry(t, htmlDoc, 3, "https://liberapay.com/example", "/assets/img/funding/liberapay.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/.profile/src/branch/%s/%s", repo.OwnerName, repo.DefaultBranch, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, ".non-diff-file-content .ui.error.message", false)
			})

			t.Run("sponsor button hidden with invalid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertNoFunding(t, htmlDoc)
			})

			t.Run("sponsor modal skips invalid funding entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42
				config["ko_fi"] = "test"
				config["liberapay"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				// invalid entries are skipped
				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")
				assertFundingEntry(t, htmlDoc, 2, "https://liberapay.com/test", "/assets/img/funding/liberapay.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Invalid type for key 'custom', expected a string or string array")
			})

			t.Run("sponsor modal skips duplicate funding entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"http://test1", "http://test1", "http://test2"}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				// duplicate entries are skipped
				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "")
				assertFundingEntry(t, htmlDoc, 2, "http://test2", "")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Duplicate entry for key 'custom': http://test1")
			})

			t.Run("funding config describes multiple issues", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42
				config["ko_fi"] = "test"
				config["whatever"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				// invalid entries are skipped
				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 1, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Errors parsing funding config:") // plural! multiple independent errors in this file!

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 2, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Invalid type for key 'custom', expected a string or string array")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding provider: whatever")
			})

			t.Run("sponsor button shown with valid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = []string{"test"}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				// no validation error!
				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				htmlDoc.AssertElement(t, ".ui.error.message", false)
			})

			t.Run("sponsor button shown with valid funding config with unknown keys", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = "test"
				config["whatever"] = "whatever"

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding provider: whatever")
			})

			t.Run("sponsor button shown with valid funding config with invalid unknown key", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"https://example.com"}
				config["ko_fi"] = "test"
				config["whatever"] = 42 // we shouldn't care how this key is shaped just yet

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding provider: whatever")
			})

			t.Run("sponsor modal shows only valid string array items", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []any{42, "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 1, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Invalid type for key 'custom', expected a string or string array")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{
					"http://test1",
					"https://example.com",
					"http://test3",
					"http://test4",
					"http://too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 4, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "")
				assertFundingEntry(t, htmlDoc, 2, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 3, "http://test3", "")
				assertFundingEntry(t, htmlDoc, 4, "http://test4", "")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Expected up to 4 of funding provider custom")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom, valid others", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "test"
				config["custom"] = []string{
					"http://test1",
					"https://example.com",
					"http://test3",
					"http://test4",
					"http://too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 5, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "")
				assertFundingEntry(t, htmlDoc, 2, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 3, "http://test3", "")
				assertFundingEntry(t, htmlDoc, 4, "http://test4", "")
				assertFundingEntry(t, htmlDoc, 5, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Expected up to 4 of funding provider custom")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom and ko_fi", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = []string{"test", "test2"}
				config["custom"] = []string{
					"http://test1",
					"https://example.com",
					"http://test3",
					"http://test4",
					"http://too_many",
				}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 5, sponsorEntries.Length())
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "")
				assertFundingEntry(t, htmlDoc, 2, "https://example.com", "")
				assertFundingEntry(t, htmlDoc, 3, "http://test3", "")
				assertFundingEntry(t, htmlDoc, 4, "http://test4", "")
				assertFundingEntry(t, htmlDoc, 5, "https://ko-fi.com/test", "/assets/img/funding/ko_fi.svg")

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Errors parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 2, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Expected up to 4 of funding provider custom")
				assert.Contains(t, fileErrorDetails.Text(), "Expected up to 1 of funding provider ko_fi")
			})

			t.Run("sponsor modal mitigates XSS", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["ko_fi"] = "\"><script>alert(1);</script><a class=\"" // URL escaped
				config["liberapay"] = "text/other" // URL escaped // TODO: Should this maybe just do without instead? When do we need to support multiple path segments here anyway?
				config["custom"] = []string{
					"#\" style=\"background: url(localhost)", // omitted (no scheme)
					"https://example.com\" class=\"rogue injection", // omitted (space in domain name)
					"https://example.com/\" class=\"rogue injection", // URL escaped
					"<script>alert`1`</script>", // omitted (no scheme)
				}

				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 3, sponsorEntries.Length())

				assertFundingEntry(t, htmlDoc, 1, "https://example.com/%22%20class=%22rogue%20injection", "")
				htmlDoc.AssertElementPredicate(t, "dialog#sponsor-modal li:nth-child(1) a", func(el *goquery.Selection) {
					assert.Equal(t, "https://example.com/\" class=\"rogue injection", el.Text())
				})

				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/%22%3E%3Cscript%3Ealert%281%29%3B%3C%2Fscript%3E%3Ca%20class=%22", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElementPredicate(t, "dialog#sponsor-modal li:nth-child(2) a", func(el *goquery.Selection) {
					assert.Equal(t, "ko-fi.com/\"><script>alert(1);</script><a class=\"", el.Text())
					assert.Zero(t, el.Children().Length()) // no real injected <script>
				})

				assertFundingEntry(t, htmlDoc, 3, "https://liberapay.com/text%2Fother", "/assets/img/funding/liberapay.svg")
				htmlDoc.AssertElementPredicate(t, "dialog#sponsor-modal li:nth-child(3) a", func(el *goquery.Selection) {
					assert.Equal(t, "liberapay.com/text/other", el.Text())
				})

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Errors parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 3, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Missing URL scheme in value for key 'custom': #%22%20style=%22background:%20url(localhost)")
				assert.Contains(t, fileErrorDetails.Text(), `Invalid URL value for key 'custom': parse "https://example.com\" class=\"rogue injection": invalid character " " in host name`)
				assert.Contains(t, fileErrorDetails.Text(), "Missing URL scheme in value for key 'custom': %3Cscript%3Ealert%601%60%3C/script%3E")
			})
		})
	}
}
