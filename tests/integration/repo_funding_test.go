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
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/assert"
)

// Returns the main HTML webpage for the given repo.
func getRepoPage(t *testing.T, repo *repo_model.Repository) *HTMLDoc {
	t.Helper()
	req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s", repo.OwnerName, repo.Name))
	resp := MakeRequest(t, req, http.StatusOK)
	return NewHTMLParser(t, resp.Body)
}

// Ensures the page's Sponsor modal has the goven number of entries.
func assertNFundingEntries(t *testing.T, htmlDoc *HTMLDoc, expectedNumberOfEntries int) {
	t.Helper()
	sponsorEntries := htmlDoc.Find("dialog#sponsor-modal li")
	assert.Equal(t, expectedNumberOfEntries, sponsorEntries.Length())
}

// Ensures the page's Sponsor modal contains the given entry.
//
// `nth` is 1-indexed, indicating which entry in the list to check.
func assertFundingEntry(t *testing.T, htmlDoc *HTMLDoc, nth uint, href, imgName string) {
	t.Helper()
	sel := fmt.Sprintf("dialog#sponsor-modal li:nth-child(%d)", nth)
	htmlDoc.AssertAttrEqual(t, sel+" a", "href", href)
	htmlDoc.AssertElement(t, sel+" svg."+imgName, true)
}

func TestSponsorButton(t *testing.T) {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	org := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 6})

	t.Run("sponsor button shown with one valid and one invalid config (first found wins)", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			config := "custom: example.com\n" + // no scheme is assumed HTTP
				"ko_fi: test"
			createFundingConfig(t, owner, repo, ".forgejo/FUNDING.yml", config) // .forgejo is checked first, so we take it whether or not it's invalid

			config = "custom: 42"
			createFundingConfig(t, owner, repo, "FUNDING.yml", config) // invalid config, but ignored anyway because we already have a .forgejo/FUNDING.yml

			htmlDoc := getRepoPage(t, repo)
			sponsorButton := htmlDoc.Find("button.sponsor")
			assert.Equal(t, 1, sponsorButton.Length())
			assert.Contains(t, sponsorButton.Text(), "Sponsor")
			htmlDoc.AssertElement(t, "button.sponsor > svg.octicon-heart", true)

			htmlDoc.AssertElement(t, "dialog#sponsor-modal", true)
			sponsorModalHeader := htmlDoc.Find("dialog#sponsor-modal header")
			assert.Equal(t, 1, sponsorModalHeader.Length())
			assert.Equal(t, fmt.Sprintf("Sponsor %s/%s", repo.OwnerName, repo.Name), strings.TrimSpace(sponsorModalHeader.Text()))

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
			htmlDoc.AssertElement(t, "button.sponsor", false)
			htmlDoc.AssertElement(t, "dialog#sponsor-modal", false)
		})
	})

	t.Run("sponsor button hidden with empty org funding config", func(t *testing.T) {
		onApplicationRun(t, func(t *testing.T, _ *url.URL) {
			mfs := forgery.MapFS{}
			mfs["FUNDING.yml"] = forgery.MapFile("\n")
			forgery.CreateRepository(t, org, &forgery.CreateRepositoryOptions{
				Name:  ".profile",
				Files: mfs,
			})

			// no entries to show!
			req := NewRequest(t, "GET", fmt.Sprintf("/%s", org.Name))
			resp := MakeRequest(t, req, http.StatusOK)
			htmlDoc := NewHTMLParser(t, resp.Body)
			htmlDoc.AssertElement(t, "button.sponsor", false)
			htmlDoc.AssertElement(t, "dialog#sponsor-modal", false)
		})
	})
}
