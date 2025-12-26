// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// WatchInfo represents an API watch status of one repository
type WatchInfo struct {
	Subscribed    bool      `json:"subscribed"`
	Ignored       bool      `json:"ignored"`
	Reason        any       `json:"reason"`
	CreatedAt     time.Time `json:"created_at"`
	URL           string    `json:"url"`
	RepositoryURL string    `json:"repository_url"`
	// WatchEvents is a bitmask of event types: 1=Issues, 2=PullRequests, 4=Releases
	// swagger:strfmt int64
	WatchEvents int64 `json:"watch_events,omitempty"`
}

// WatchOptions represents options for watching a repository
type WatchOptions struct {
	// WatchEvents is a bitmask of event types to watch: 1=Issues, 2=PullRequests, 4=Releases
	// If not specified, uses user/instance defaults (typically 7 = all events)
	WatchEvents *int64 `json:"watch_events,omitempty"`
}
