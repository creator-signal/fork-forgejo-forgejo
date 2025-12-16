// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// CreateRemoteRegistryOption options for creating a RemoteRegistry
// swagger:model
type CreateRemoteRegistryOption struct {
	// required: true
	Name string `json:"name" binding:"Required"`
	// required: true
	RemoteType string `json:"remote_type" binding:"Required"`
	// required: true
	RemoteURL      string `json:"remote_url" binding:"Required;ValidUrl;MaxSize(255)"`
	RemoteUser     string `json:"remote_user"`
	RemotePassword string `json:"remote_pass"`
	RemoteToken    string `json:"remote_token"`
}
