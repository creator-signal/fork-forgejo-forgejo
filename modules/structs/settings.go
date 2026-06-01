// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// GeneralRepoSettings contains global repository settings exposed by API
type GeneralRepoSettings struct {
	MirrorsDisabled      bool `json:"mirrors_disabled"`
	HTTPGitDisabled      bool `json:"http_git_disabled"`
	MigrationsDisabled   bool `json:"migrations_disabled"`
	StarsDisabled        bool `json:"stars_disabled"`
	ForksDisabled        bool `json:"forks_disabled"`
	TimeTrackingDisabled bool `json:"time_tracking_disabled"`
	LFSDisabled          bool `json:"lfs_disabled"`
}

// GeneralUISettings contains global ui settings exposed by API
type GeneralUISettings struct {
	DefaultTheme     string   `json:"default_theme"`
	AllowedReactions []string `json:"allowed_reactions"`
	CustomEmojis     []string `json:"custom_emojis"`
}

// GeneralAPISettings contains global api settings exposed by it
type GeneralAPISettings struct {
	MaxResponseItems       int   `json:"max_response_items"`
	DefaultPagingNum       int   `json:"default_paging_num"`
	DefaultGitTreesPerPage int   `json:"default_git_trees_per_page"`
	DefaultMaxBlobSize     int64 `json:"default_max_blob_size"`
}

// GeneralAttachmentSettings contains global Attachment settings exposed by API
type GeneralAttachmentSettings struct {
	Enabled      bool   `json:"enabled"`
	AllowedTypes string `json:"allowed_types"`
	MaxSize      int64  `json:"max_size"`
	MaxFiles     int    `json:"max_files"`
}

// FundingProvider contains a funding provider exposed by API
type FundingProvider struct {
	// Identifies the funding provider
	Name  string `json:"name"`
	// The max number of times a funding config may specify this provider
	Limit uint   `json:"limit"`
	// A format string for link text for an instance of this provider
	Text  string `json:"text"`
	// A format string for the URL to a profile on this provider
	URL   string `json:"url"`
	// Path to the logo of the funding provider
	Icon  string `json:"icon"`
	// Path to the dark-theme logo, if any, of the funding provider
	IconDark string `json:"icon_dark"`
}

// FundingProvider contains global funding provider settings exposed by API
type FundingSettings struct {
	Providers []*FundingProvider `json:"providers"`
}
