// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package container

import (
	user_model "forgejo.org/models/user"
)

// ManifestCreationInfo describes a manifest to create
type ManifestCreationInfo struct {
	MediaType  string
	Owner      *user_model.User
	Creator    *user_model.User
	Image      string
	Reference  string
	IsTagged   bool
	Properties map[string]string
}
