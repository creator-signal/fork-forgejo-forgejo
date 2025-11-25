// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package flatpak

import (
	"fmt"
	"strings"

	"forgejo.org/modules/setting"
)

type FlatpakKind string

const (
	FlatpakKindApp     FlatpakKind = "app"
	FlatpakKindRuntime FlatpakKind = "runtime"
)

type Flatpak struct {
	ID          string      `json:"id,omitempty"`
	Kind        FlatpakKind `json:"kind,omitempty"`
	Branch      string      `json:"branch,omitempty"`
	RuntimeRepo string      `json:"runtime_repo,omitempty"`
}

// Returns the name for the remote
func GetRepoName(username string) string {
	return fmt.Sprintf("%s-%s", strings.ToLower(setting.AppName), strings.ToLower(username))
}

// Returns the oci+ URL for the repo
func GetRepoURL(username string) string {
	return fmt.Sprintf("oci+%sapi/packages/%s/container", setting.AppURL, strings.ToLower(username))
}

// Returns the URL to the .flatpakrepo file
func GetRepoInfoURL(username string) string {
	return fmt.Sprintf("%sapi/packages/%s/container/flatpak/repo.flatpakrepo", setting.AppURL, strings.ToLower(username))
}
