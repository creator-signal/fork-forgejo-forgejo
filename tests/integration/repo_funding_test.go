// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TODO: link to the page from the modal when there are errors
// TODO: handle instance-custom entries (app.ini)
// TODO: the given values are interpolated and escaped correctly; a repo can't simply cause XSS using FUNDING.yml! (Go templates and translations should be smart enough for that, but we should add a test to be sure)
// TODO: test uniqueness
// TODO: Test admin config with a provider with limit of 0
// TODO: Test admin config overriding a provider limit

func TestSponsorButton(t *testing.T) {
	// TODO: also test against a user profile
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	t.Run("sponsor button hidden without funding config (repo)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			sponsorButton := htmlDoc.Find("button.sponsor")
			assert.Equal(t, 0, sponsorButton.Length())

			sponsorModal := htmlDoc.Find("dialog#sponsor-modal")
			assert.Equal(t, 0, sponsorModal.Length())
		})
	})

	t.Run("sponsor button hidden without funding config (user)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", fmt.Sprintf("/%s", repo.OwnerName))
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			sponsorButton := htmlDoc.Find("button.sponsor")
			assert.Equal(t, 0, sponsorButton.Length())

			sponsorModal := htmlDoc.Find("dialog#sponsor-modal")
			assert.Equal(t, 0, sponsorModal.Length())
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

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			sponsorButton := htmlDoc.Find("button.sponsor")
			assert.Equal(t, 1, sponsorButton.Length())
			assert.Contains(t, sponsorButton.Text(), "Sponsor")
			htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

			// e2e tests check open/close behavior and accessibility, here we check data
			sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
			assert.Equal(t, 1, sponsorModalHeader.Length())
			assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

			sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
			assert.Equal(t, 2, sponsorEntries.Length())

			htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://example.com")
			htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
			htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

			htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://ko-fi.com/test")
			htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) img", "src", "/assets/img/funding/ko_fi.svg")
			htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", false)
		})
	})

	for _, treePath := range testFundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			t.Run("sponsor button hidden with invalid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 0, sponsorButton.Length())

				sponsorModal := htmlDoc.Find("dialog#sponsor-modal")
				assert.Equal(t, 0, sponsorModal.Length())
			})

			t.Run("sponsor modal skips invalid funding entries", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42
				config["ko_fi"] = "test"
				config["liberapay"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())

				// invalid entries are skipped
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", false)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://liberapay.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) img", "src", "/assets/img/funding/liberapay.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "custom has an invalid type. Expected string or string array")
			})

			t.Run("funding config describes multiple issues", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42
				config["ko_fi"] = "test"
				config["whatever"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 1, sponsorEntries.Length())

				// invalid entries are skipped
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Errors parsing funding config:") // plural! multiple independent errors in this file!

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 2, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "custom has an invalid type. Expected string or string array")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding platform: whatever")
			})

			t.Run("sponsor button shown with valid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = []string{"test"}

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", false)

				// no validation error!
				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

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

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding platform: whatever")
			})

			t.Run("sponsor button shown with valid funding config with invalid unknown key", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"https://example.com"}
				config["ko_fi"] = "test"
				config["whatever"] = 42 // we shouldn't care how this key is shaped just yet

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 2, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "Unknown funding platform: whatever")
			})

			t.Run("sponsor modal shows only valid string array items", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []any{42, "https://example.com"}

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 1, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

				htmlDoc = NewHTMLParser(t, resp.Body)
				fileError := htmlDoc.Find(".non-diff-file-content .ui.error.message")
				assert.Equal(t, 1, fileError.Length())
				assert.Contains(t, fileError.Text(), "Error parsing funding config:")

				fileErrorDetails := htmlDoc.Find(".ui.error.message li")
				assert.Equal(t, 1, fileErrorDetails.Length())
				assert.NotContains(t, fileErrorDetails.Text(), "Unknown error")
				assert.Contains(t, fileErrorDetails.Text(), "custom has an invalid type. Expected string or string array")
			})

			t.Run("sponsor modal shows only up to the configured limit for custom", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 4, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "test1")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(3) a", "href", "test3")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(4) a", "href", "test4")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) svg.octicon-link", true)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

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
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 5, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "test1")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(3) a", "href", "test3")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(4) a", "href", "test4")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(5) a", "href", "https://ko-fi.com/test")
			htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(5) img", "src", "/assets/img/funding/ko_fi.svg")
			htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(5) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

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
				config["custom"] = []string{"test1", "https://example.com", "test3", "test4", "too_many"}

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button.sponsor")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

				sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
				assert.Equal(t, 1, sponsorModalHeader.Length())
				assert.Contains(t, sponsorModalHeader.Text(), fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name))

				sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
				assert.Equal(t, 5, sponsorEntries.Length())

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(1) a", "href", "test1")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(1) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(2) a", "href", "https://example.com")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(2) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(3) a", "href", "test3")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(3) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(4) a", "href", "test4")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) img", false)
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(4) svg.octicon-link", true)

				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(5) a", "href", "https://ko-fi.com/test")
				htmlDoc.AssertAttrEqual(t, "dialog#sponsor-modal li:nth-child(5) img", "src", "/assets/img/funding/ko_fi.svg")
				htmlDoc.AssertElement(t, "dialog#sponsor-modal li:nth-child(5) svg.octicon-link", false)

				req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/%s", repo.OwnerName, repo.Name, treePath))
				resp = MakeRequest(t, req, http.StatusOK)

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
		})
	}
}
