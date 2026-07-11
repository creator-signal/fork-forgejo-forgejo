// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

// Package homepage contains logic for the landing page.
package homepage

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	"go.yaml.in/yaml/v3"
)

// Section type identifiers accepted as a section's "type" in home.yml.
const (
	SectionRepos = "repositories"
	SectionOrgs  = "organizations"
	SectionStats = "statistics"
	SectionLinks = "links"
)

// Config is a parsed home.yml.
type Config struct {
	Hero     Hero       `yaml:"hero"`
	NavLinks []*NavLink `yaml:"nav_links"`
	Sections []*Section `yaml:"sections"`
}

// NavLink is for extra navbar links on the landing page.
type NavLink struct {
	Text   string `yaml:"text"`
	URL    string `yaml:"url"`
	Icon   string `yaml:"icon"`
	Button bool   `yaml:"button"`
}

// Hero is the configuration for the hero section.
type Hero struct {
	Title       string    `yaml:"title"`
	Sub         string    `yaml:"sub"`
	Description string    `yaml:"description"`
	Logo        *bool     `yaml:"logo"`
	PoweredBy   *bool     `yaml:"powered_by"`
	Buttons     []*Button `yaml:"buttons"`
}

// ShowLogo reports whether the hero logo should be rendered.
func (h Hero) ShowLogo() bool {
	return h.Logo == nil || *h.Logo
}

// ShowPoweredBy reports whether the "Powered by Forgejo" credit should be shown.
func (h Hero) ShowPoweredBy() bool {
	return h.PoweredBy == nil || *h.PoweredBy
}

// Button is a hero call-to-action.
type Button struct {
	Text    string `yaml:"text"`
	URL     string `yaml:"url"`
	Primary bool   `yaml:"primary"`
}

// Section is one landing-page block.
type Section struct {
	Type  string `yaml:"type"`
	Title string `yaml:"title"`

	// repositories
	Sort string `yaml:"sort"`

	// statistics
	Items []string `yaml:"items"`

	// links
	Links []*Link `yaml:"links"`
}

// Link is an entry of a links section.
type Link struct {
	Text        string `yaml:"text"`
	URL         string `yaml:"url"`
	Icon        string `yaml:"icon"`
	Description string `yaml:"description"`
	Button      string `yaml:"button"`
}

func defaultConfig() *Config {
	return &Config{
		Sections: []*Section{
			{Type: SectionStats, Items: []string{"repositories", "users", "organizations"}},
			{Type: SectionRepos, Sort: "stars"},
			{Type: SectionOrgs},
		},
	}
}

var config atomic.Pointer[Config]

// Init reads home.yml.
func Init() {
	config.Store(read())
}

// Get returns the configuration loaded by Init.
func Get() *Config {
	return config.Load()
}

func configPath() string {
	if setting.Home.ConfigFile != "" {
		return setting.Home.ConfigFile
	}
	return filepath.Join(setting.CustomPath, "home.yml")
}

func read() *Config {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Error("Unable to read %s: %v", path, err)
		}
		return defaultConfig()
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		log.Error("Unable to parse %s: %v", path, err)
		return defaultConfig()
	}
	return cfg
}
