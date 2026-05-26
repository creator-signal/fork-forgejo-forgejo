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

// TODO: Based on other UI integration tests, ensure the following:
// TODO:  the correct icon and text display for all modal parts (built-in and instance-custom entries)
// TODO:  unknown payment providers are omitted from the UI
// TODO:  the given values are interpolated and escaped correctly; a repo can't simply cause XSS using FUNDING.yml
// TODO:  (e2e?) selecting the sponsor button opens a dialog that shows the info
// TODO:  figure out when profile_big_avatar is shown, and test sponsor-button there too

func TestSponsorButton(t *testing.T) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	t.Run("sponsor button hidden without funding config", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
			resp := MakeRequest(t, req, http.StatusOK)

			htmlDoc := NewHTMLParser(t, resp.Body)
			sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
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
			sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
			assert.Equal(t, 1, sponsorButton.Length())
			assert.Contains(t, sponsorButton.Text(), "Sponsor")
			htmlDoc.AssertElement(t, "button[data-test='sponsor-button'] > svg.octicon-heart", true)

			// TODO: check for corresponding dialog
		})
	})

	for _, treePath := range fundingCandidates {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			t.Run("sponsor button hidden with invalid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = 42
				config["ko_fi"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
				assert.Equal(t, 0, sponsorButton.Length())

				sponsorModal := htmlDoc.Find("dialog#sponsor-modal")
				assert.Equal(t, 0, sponsorModal.Length())
			})

			t.Run("sponsor button shown with valid funding config", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = "test"

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button[data-test='sponsor-button'] > svg.octicon-heart", true)

				// TODO: check for corresponding dialog
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
				sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button[data-test='sponsor-button'] > svg.octicon-heart", true)

				// TODO: check for corresponding dialog, and that the funding renderer contains relevant error info
			})

			t.Run("sponsor button shown with valid funding config with invalid unknown key", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				config := make(map[string]any)
				config["custom"] = "https://example.com"
				config["ko_fi"] = "test"
				config["whatever"] = 42 // we shouldn't care how this key is shaped just yet

				createFundingConfig(t, owner, repo, treePath, config)

				req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
				resp := MakeRequest(t, req, http.StatusOK)

				htmlDoc := NewHTMLParser(t, resp.Body)
				sponsorButton := htmlDoc.Find("button[data-test='sponsor-button']")
				assert.Equal(t, 1, sponsorButton.Length())
				assert.Contains(t, sponsorButton.Text(), "Sponsor")
				htmlDoc.AssertElement(t, "button[data-test='sponsor-button'] > svg.octicon-heart", true)

				// TODO: check for corresponding dialog, and that the funding renderer contains relevant error info
			})
		})
	}
}
