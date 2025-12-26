// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

const (
	// SettingsKeyHiddenCommentTypes is the setting key for hidden comment types
	SettingsKeyHiddenCommentTypes = "issue.hidden_comment_types"
	// SettingsKeyDiffWhitespaceBehavior is the setting key for whitespace behavior of diff
	SettingsKeyDiffWhitespaceBehavior = "diff.whitespace_behaviour"
	// SettingsKeyShowOutdatedComments is the setting key whether or not to show outdated comments in PRs
	SettingsKeyShowOutdatedComments = "comment_code.show_outdated"
	// SettingsKeyDefaultWatchEvents is the setting key for default watch events when watching a repository
	SettingsKeyDefaultWatchEvents = "repo.default_watch_events"
	// SettingsKeyAutoWatchOnCreate is the setting key for auto-watch when creating a repo
	SettingsKeyAutoWatchOnCreate = "repo.auto_watch_on_create"
	// SettingsKeyAutoWatchOnAccess is the setting key for auto-watch when getting access to a repo
	SettingsKeyAutoWatchOnAccess = "repo.auto_watch_on_access"
	// SettingsKeyAutoWatchOnContribute is the setting key for auto-watch on first contribution
	SettingsKeyAutoWatchOnContribute = "repo.auto_watch_on_contribute"
	// UserActivityPubPrivPem is user's private key
	UserActivityPubPrivPem = "activitypub.priv_pem"
	// UserActivityPubPubPem is user's public key
	UserActivityPubPubPem = "activitypub.pub_pem"
)
