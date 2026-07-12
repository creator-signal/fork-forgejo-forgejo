// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.org/modules/homepage"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func useHomeConfig(t *testing.T, yaml string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "home.yml")
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))
	reset := test.MockVariableValue(&setting.Home.ConfigFile, path)
	homepage.Init()
	t.Cleanup(func() {
		reset()
		homepage.Init()
	})
}

func TestHomeCustomConfig(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	yaml := `
hero:
  title: Integration Forge
  sub: welcome
nav_links:
  - {text: DonateNav, url: https://example.com/donate, button: true}
sections:
  - type: statistics
    items: [repositories, users]
  - type: repositories
    title: Popular things
    sort: stars
  - type: links
    title: Resources
    links:
      - {text: DocsLink, url: https://example.com/docs, description: read the docs}
`
	useHomeConfig(t, yaml)

	req := NewRequest(t, "GET", "/")
	resp := emptyTestSession(t).MakeRequest(t, req, http.StatusOK)
	body := resp.Body.String()
	htmlDoc := NewHTMLParser(t, resp.Body)

	assert.Equal(t, "Integration Forge", strings.TrimSpace(htmlDoc.Find("h1.home-hero-title").Text()))

	assert.Positive(t, htmlDoc.Find("#navbar a.item.primary").Length())
	assert.Contains(t, body, "DonateNav")

	assert.Positive(t, htmlDoc.Find(".home-stats .home-stat").Length())

	assert.Contains(t, body, "Popular things")
	assert.Positive(t, htmlDoc.Find(".home-card-grid a.home-card").Length())
	assert.Positive(t, htmlDoc.Find(".home-view-all a").Length())

	assert.Contains(t, body, "Resources")
	assert.Contains(t, body, "DocsLink")
}

func TestHomeCustomConfigRequireSignInViewHidesSections(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	yaml := `
sections:
  - type: statistics
    items: [repositories, users]
`
	useHomeConfig(t, yaml)

	defer test.MockVariableValue(&setting.Service.RequireSignInView, true)()

	req := NewRequest(t, "GET", "/")
	resp := emptyTestSession(t).MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	assert.Zero(t, htmlDoc.Find(".home-section").Length())
	assert.Zero(t, htmlDoc.Find(".home-stats .home-stat").Length())
}

func TestHomeCustomConfigNavIsHomeOnly(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	useHomeConfig(t, "nav_links:\n  - {text: DonateNav, url: https://example.com/donate}\n")

	session := emptyTestSession(t)

	home := session.MakeRequest(t, NewRequest(t, "GET", "/"), http.StatusOK)
	assert.Contains(t, home.Body.String(), "DonateNav")

	explore := session.MakeRequest(t, NewRequest(t, "GET", "/explore/repos"), http.StatusOK)
	assert.NotContains(t, explore.Body.String(), "DonateNav")
}
