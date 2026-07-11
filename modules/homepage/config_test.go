// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package homepage

import (
	"os"
	"path/filepath"
	"testing"

	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeConfig(t *testing.T, contents string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "home.yml")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o644))
	setting.Home.ConfigFile = path
}

func TestReadDefaultWhenAbsent(t *testing.T) {
	setting.Home.ConfigFile = filepath.Join(t.TempDir(), "does-not-exist.yml")

	cfg := read()
	require.Len(t, cfg.Sections, 3)
	assert.Equal(t, SectionStats, cfg.Sections[0].Type)
	assert.Equal(t, SectionRepos, cfg.Sections[1].Type)
	assert.Equal(t, SectionOrgs, cfg.Sections[2].Type)
}

func TestReadParsesConfig(t *testing.T) {
	writeConfig(t, `
hero:
  title: My Forge
  sub: A tagline
  description: A longer description
  logo: false
  powered_by: false
  buttons:
    - {text: Sign up, url: /user/sign_up, primary: true}
    - {text: Learn more, url: https://example.com}
nav_links:
  - {text: Docs, url: https://example.com/docs, icon: book}
  - {text: Donate, url: https://example.com/donate, button: true}
sections:
  - type: statistics
    items: [repositories, users]
  - type: repositories
    title: Popular
    sort: stars
  - type: links
    links:
      - {text: Chat, url: https://example.com/chat, icon: comment-discussion, description: Say hi, button: Join}
`)

	cfg := read()

	assert.Equal(t, "My Forge", cfg.Hero.Title)
	assert.Equal(t, "A tagline", cfg.Hero.Sub)
	assert.Equal(t, "A longer description", cfg.Hero.Description)
	assert.False(t, cfg.Hero.ShowLogo())
	assert.False(t, cfg.Hero.ShowPoweredBy())

	require.Len(t, cfg.Hero.Buttons, 2)
	assert.Equal(t, "Sign up", cfg.Hero.Buttons[0].Text)
	assert.True(t, cfg.Hero.Buttons[0].Primary)
	assert.False(t, cfg.Hero.Buttons[1].Primary)

	require.Len(t, cfg.NavLinks, 2)
	assert.Equal(t, "book", cfg.NavLinks[0].Icon)
	assert.False(t, cfg.NavLinks[0].Button)
	assert.True(t, cfg.NavLinks[1].Button)

	require.Len(t, cfg.Sections, 3)
	assert.Equal(t, []string{"repositories", "users"}, cfg.Sections[0].Items)

	repos := cfg.Sections[1]
	assert.Equal(t, "Popular", repos.Title)
	assert.Equal(t, "stars", repos.Sort)

	links := cfg.Sections[2]
	require.Len(t, links.Links, 1)
	assert.Equal(t, "Join", links.Links[0].Button)
	assert.Equal(t, "Say hi", links.Links[0].Description)
}

func TestReadInvalidYAMLFallsBackToDefault(t *testing.T) {
	writeConfig(t, "nav_links: [this is not: valid yaml")

	cfg := read()
	require.Len(t, cfg.Sections, 3)
	assert.Equal(t, SectionStats, cfg.Sections[0].Type)
}

func TestHeroToggleDefaults(t *testing.T) {
	var h Hero
	assert.True(t, h.ShowLogo())
	assert.True(t, h.ShowPoweredBy())

	no := false
	h = Hero{Logo: &no, PoweredBy: &no}
	assert.False(t, h.ShowLogo())
	assert.False(t, h.ShowPoweredBy())
}

func TestInitAndGet(t *testing.T) {
	writeConfig(t, "hero:\n  title: Loaded\n")
	Init()

	cfg := Get()
	assert.Equal(t, "Loaded", cfg.Hero.Title)
	assert.Same(t, cfg, Get())

	writeConfig(t, "hero:\n  title: Reloaded\n")
	Init()
	assert.Equal(t, "Reloaded", Get().Hero.Title)
}
