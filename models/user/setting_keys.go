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
	// UserActivityPubPrivPem is user's private key
	UserActivityPubPrivPem = "activitypub.priv_pem"
	// UserActivityPubPubPem is user's public key
	UserActivityPubPubPem = "activitypub.pub_pem"
	// SettingsKeyFirstDoW is the setting key for first day of week (0=Sunday, 1=Monday, 5=Friday, 6=Saturday)
	SettingsKeyFirstDoW = "first_day_of_week"
)
