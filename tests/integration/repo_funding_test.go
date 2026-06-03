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

// Returns the main HTML webpage for the given file in the given repo.
func getFilePage(t *testing.T, repo *repo_model.Repository, treePath string) *HTMLDoc {
	t.Helper()

	req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/%s/%s", repo.OwnerName, repo.Name, repo.DefaultBranch, treePath))
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

// Ensures the page's Sponsor modal has the given header text.
func assertSponsorModalHeader(t *testing.T, htmlDoc *HTMLDoc, expectedHeaderText string) {
	htmlDoc.AssertElement(t, "dialog#sponsor-modal", true)
	sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
	assert.Equal(t, 1, sponsorModalHeader.Length())
	assert.Equal(t, expectedHeaderText, strings.TrimSpace(sponsorModalHeader.Text()))
}

// Ensures the page's Sponsor modal has the goven number of entries.
func assertNFundingEntries(t *testing.T, htmlDoc *HTMLDoc, expectedNumberOfEntries int) {
	sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
	assert.Equal(t, expectedNumberOfEntries, sponsorEntries.Length())
}

// Ensures the page's Sponsor modal contains the given entry.
//
// `nth` is 1-indexed, indicating which entry in the list to check.
func assertFundingEntry(t *testing.T, htmlDoc *HTMLDoc, nth uint, href, imgName string) {
	sel := fmt.Sprintf("dialog#sponsor-modal li:nth-child(%d)", nth)

	htmlDoc.AssertAttrEqual(t, sel+" a", "href", href)
	htmlDoc.AssertElement(t, sel+" svg."+imgName, true)
}

// Ensures the page's Sponsor modal contains the given text in the given entry,
// and that the element has no inner child elements.
//
// `nth` is 1-indexed, indicating which entry in the list to check.
func assertFundingEntryHasText(t *testing.T, htmlDoc *HTMLDoc, nth uint, expectedText string) {
	sel := fmt.Sprintf("dialog#sponsor-modal li:nth-child(%d)", nth)

	htmlDoc.AssertElementPredicate(t, sel+" a", func(el *goquery.Selection) {
		assert.Equal(t, expectedText, el.Text())
		assert.Zero(t, el.Children().Length()) // no injected <script>, etc.
	})
}

// Ensures the page contains the given number of file errors.
func assertNFundingErrors(t *testing.T, htmlDoc *HTMLDoc, expectedNumberOfErrors int) {
	if expectedNumberOfErrors == 0 {
		htmlDoc.AssertElement(t, ".ui.error.message", false)
		return
	}

	fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
	assert.Equal(t, 1, fileError.Length())
	if expectedNumberOfErrors == 1 {
		assert.Contains(t, strings.TrimSpace(fileError.Text()), "Error parsing funding config:")
	} else {
		assert.Contains(t, strings.TrimSpace(fileError.Text()), "Errors parsing funding config:")
	}

	fileErrorDetails := htmlDoc.Find(".non-diff-file-content .ui.error.message li")
	assert.Equal(t, expectedNumberOfErrors, fileErrorDetails.Length())
}

// Ensures the page's file error details contain the given notice.
//
// `nth` is 1-indexed, indicating which error detail in the list to check.
func assertFundingError(t *testing.T, htmlDoc *HTMLDoc, nth int, expectedText string) {
	fileErrorDetails := htmlDoc.Find(".non-diff-file-content .ui.error.message li")
	assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")

	sel := fmt.Sprintf(".non-diff-file-content .ui.error.message li:nth-child(%d)", nth)
	fileErrorDetails = htmlDoc.Find(sel)
	assert.Equal(t, expectedText, strings.TrimSpace(fileErrorDetails.Text()))
}

// e2e tests check open/close behavior and accessibility, here we check data
func TestSponsorButton(t *testing.T) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 6})

	t.Run("sponsor button hidden without funding config (repo)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			htmlDoc := getRepoPage(t, repo)
			assertNoFunding(t, htmlDoc)
		})
	})

	t.Run("sponsor button hidden without funding config (user)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			htmlDoc := getUserPage(t, owner)
			assertNoFunding(t, htmlDoc)
		})
	})

	t.Run("sponsor button hidden without funding config (org)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			htmlDoc := getUserPage(t, org)
			assertNoFunding(t, htmlDoc)
		})
	})

	t.Run("sponsor button shown with one valid and one invalid config (first found wins)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			config := "custom: example.com\n" + // no scheme is assumed HTTP
				"ko_fi: test"
			createFundingConfig(t, owner, repo, ".forgejo/FUNDING.yml", config) // .forgejo is checked first, so we take it whether or not it's invalid

			config = "custom: 42"
			createFundingConfig(t, owner, repo, "FUNDING.yml", config) // invalid config, but ignored anyway because we already have a .forgejo/FUNDING.yml

			htmlDoc := getRepoPage(t, repo)
			assertSponsorButton(t, htmlDoc)
			assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
			assertNFundingEntries(t, htmlDoc, 2)
			assertFundingEntry(t, htmlDoc, 1, "http://example.com", "octicon-link")
			assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "brand-ko_fi")
		})
	})

	t.Run("sponsor button shown with one invalid and one valid config (first found wins)", func(t *testing.T) {
		// reversed files from the above test: the one checked first is invalid in this case
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			config := "custom: 42"
			createFundingConfig(t, owner, repo, ".forgejo/FUNDING.yml", config) // .forgejo is checked first, but this one is invalid. we take it anyway, because we assume it's intentional

			config = "custom: example.com\n" +
				"ko_fi: test"
			createFundingConfig(t, owner, repo, "FUNDING.yml", config) // valid config, but ignored because we already have a .forgejo/FUNDING.yml

			htmlDoc := getRepoPage(t, repo)
			assertNoFunding(t, htmlDoc)
		})
	})

	// FIXME: we don't need to integration-test every permutation like this, do we? and some of these would probably make be much unit tests. Consider TestAPIRepoIssueConfigPaths, TestAPIRepoValidateIssueConfig, and TestAPIRepoGetIssueConfig for example
	for _, treePath := range testFundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			t.Run("sponsor button hidden with empty funding config (repo)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				createFundingConfig(t, owner, repo, treePath, ``)

				htmlDoc := getRepoPage(t, repo)
				assertNoFunding(t, htmlDoc) // no entries to show!
			})

			t.Run("sponsor button hidden with empty funding config (user)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile("\n")
				forgery.CreateRepository(t, owner, &forgery.CreateRepositoryOptions{
					Name:  ".profile",
					Files: mfs,
				})

				htmlDoc := getUserPage(t, owner)
				assertNoFunding(t, htmlDoc) // no entries to show!
			})

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
custom: example.com
`)
				repo := forgery.CreateRepository(t, user, &forgery.CreateRepositoryOptions{
					Name:  ".profile",
					Files: mfs,
				})

				// includes the whole config, in the order it was given
				htmlDoc = getUserPage(t, user)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s", user.Name))
				assertNFundingEntries(t, htmlDoc, 3)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/example", "brand-ko_fi")
				assertFundingEntry(t, htmlDoc, 2, "https://liberapay.com/example", "brand-liberapay")
				assertFundingEntry(t, htmlDoc, 3, "http://example.com", "octicon-link")

				// no validation error!
				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 0)
			})

			t.Run("sponsor button hidden with empty funding config (org)", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile("\n")
				forgery.CreateRepository(t, org, &forgery.CreateRepositoryOptions{
					Name:  ".profile",
					Files: mfs,
				})

				htmlDoc := getUserPage(t, org)
				assertNoFunding(t, htmlDoc) // no entries to show!
			})

			t.Run("sponsor button shown on org profile", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				mfs := forgery.MapFS{}
				mfs[treePath] = forgery.MapFile(`
ko_fi: example
liberapay: example
custom: example.com
`)
				repo := forgery.CreateRepository(t, org, &forgery.CreateRepositoryOptions{
					Name:  ".profile",
					Files: mfs,
				})

				// includes the whole config, in the order it was given
				htmlDoc := getUserPage(t, org)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s", org.FullName))
				assertNFundingEntries(t, htmlDoc, 3)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/example", "brand-ko_fi")
				assertFundingEntry(t, htmlDoc, 2, "https://liberapay.com/example", "brand-liberapay")
				assertFundingEntry(t, htmlDoc, 3, "http://example.com", "octicon-link")

				// no validation error!
				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 0)
			})

			t.Run("sponsor button hidden with invalid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: 42`
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertNoFunding(t, htmlDoc) // no valid entries
			})

			t.Run("sponsor modal skips invalid funding entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom: 42\n" +
					"ko_fi: test\n" +
					"liberapay: test"
				createFundingConfig(t, owner, repo, treePath, config)

				// invalid entries are skipped
				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "brand-ko_fi")
				assertFundingEntry(t, htmlDoc, 2, "https://liberapay.com/test", "brand-liberapay")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Invalid type for key 'custom', expected a string or string array")
			})

			t.Run("sponsor modal skips duplicate funding entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom: [test1, test1, test2]"
				createFundingConfig(t, owner, repo, treePath, config)

				// duplicate entries are skipped
				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "http://test2", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Duplicate entry for key 'custom': http://test1")
			})

			t.Run("sponsor modal skips duplicate funding entries where one has a scheme", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [test1, "http://test1", test2]`
				createFundingConfig(t, owner, repo, treePath, config)

				// duplicate entries are skipped
				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "http://test2", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Duplicate entry for key 'custom': http://test1")
			})

			t.Run("funding config describes multiple issues", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom: 42\n" +
					"ko_fi: test\n" +
					"whatever: test"
				createFundingConfig(t, owner, repo, treePath, config)

				// invalid entries are skipped
				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 1)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "brand-ko_fi")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 2)
				assertFundingError(t, htmlDoc, 1, "Invalid type for key 'custom', expected a string or string array")
				assertFundingError(t, htmlDoc, 2, "Unknown funding provider: whatever")
			})

			t.Run("sponsor button shown with valid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: "https://example.com"` + "\n" +
					"patreon: [test]"
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "https://patreon.com/test", "brand-patreon")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 0)
			})

			t.Run("sponsor button shown with valid funding config with unknown keys", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: "https://example.com"` + "\n" +
					"ko_fi: test\n" +
					"whatever: whatever"
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "brand-ko_fi")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Unknown funding provider: whatever")
			})

			t.Run("sponsor button shown with valid funding config with invalid unknown key", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: ["https://example.com"]` + "\n" + // single-element list is acceptable
					"ko_fi: test\n" +
					"whatever: 42" // we shouldn't care how this key is shaped just yet
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 2)
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "https://ko-fi.com/test", "brand-ko_fi")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Unknown funding provider: whatever")
			})

			t.Run("sponsor modal shows only valid string array items", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := `custom: [42, "https://example.com"]`
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 1)
				assertFundingEntry(t, htmlDoc, 1, "https://example.com", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Invalid type for key 'custom', expected a string or string array")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 4)
				assertFundingEntry(t, htmlDoc, 1, "http://test1", "octicon-link")
				assertFundingEntry(t, htmlDoc, 2, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 3, "http://test3", "octicon-link")
				assertFundingEntry(t, htmlDoc, 4, "http://test4", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Expected up to 4 of funding provider custom")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom, valid others", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: test\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 5)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "brand-ko_fi")
				assertFundingEntry(t, htmlDoc, 2, "http://test1", "octicon-link")
				assertFundingEntry(t, htmlDoc, 3, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 4, "http://test3", "octicon-link")
				assertFundingEntry(t, htmlDoc, 5, "http://test4", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 1)
				assertFundingError(t, htmlDoc, 1, "Expected up to 4 of funding provider custom")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom and ko_fi", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := "ko_fi: [test, test2]\n" +
					"tidelift: 'npm/example'\n" +
					"custom:\n" +
					"- test1\n" +
					`- "https://example.com"` + "\n" +
					"- test3\n" +
					"- test4\n" +
					"- too_many"
				createFundingConfig(t, owner, repo, treePath, config)

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 5)
				assertFundingEntry(t, htmlDoc, 1, "https://ko-fi.com/test", "brand-ko_fi")
				assertFundingEntry(t, htmlDoc, 2, "http://test1", "octicon-link")
				assertFundingEntry(t, htmlDoc, 3, "https://example.com", "octicon-link")
				assertFundingEntry(t, htmlDoc, 4, "http://test3", "octicon-link")
				assertFundingEntry(t, htmlDoc, 5, "http://test4", "octicon-link")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 3)
				assertFundingError(t, htmlDoc, 1, "Expected up to 1 of funding provider ko_fi")
				assertFundingError(t, htmlDoc, 2, "Funding provider tidelift is not allowed")
				assertFundingError(t, htmlDoc, 3, "Expected up to 4 of funding provider custom")
			})

			t.Run("sponsor modal mitigates XSS", func(t *testing.T) {
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

				htmlDoc := getRepoPage(t, repo)
				assertSponsorButton(t, htmlDoc)
				assertSponsorModalHeader(t, htmlDoc, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))
				assertNFundingEntries(t, htmlDoc, 3)

				assertFundingEntry(t, htmlDoc, 1, "http://#%22%20style=%22background:%20url%28localhost%29", "octicon-link")
				assertFundingEntryHasText(t, htmlDoc, 1, "#\" style=\"background: url(localhost)")

				assertFundingEntry(t, htmlDoc, 2, "https://example.com/%22%20class=%22rogue%20injection", "octicon-link")
				assertFundingEntryHasText(t, htmlDoc, 2, "https://example.com/\" class=\"rogue injection")

				assertFundingEntry(t, htmlDoc, 3, "http://%3Cscript%3Ealert%601%60%3C/script%3E", "octicon-link")
				assertFundingEntryHasText(t, htmlDoc, 3, "<script>alert`1`</script>")

				htmlDoc = getFilePage(t, repo, treePath)
				assertNFundingErrors(t, htmlDoc, 4)
				assertFundingError(t, htmlDoc, 1, "Value for key 'ko_fi' does not match pattern /^[^/]+$/")
				assertFundingError(t, htmlDoc, 2, "Value for key 'liberapay' does not match pattern /^[^/]+$/")
				assertFundingError(t, htmlDoc, 3, `Value for key 'thanks_dev' does not match pattern /^[^/]+\/[^/]+\/[^/]+$/`)
				assertFundingError(t, htmlDoc, 4, `Invalid URL value for key 'custom': parse "https://example.com\" class=\"rogue injection": invalid character " " in host name`)
			})
		})
	}
}
