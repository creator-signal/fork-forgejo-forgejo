// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// CreateOrgOption options for creating an organization
// swagger:model
type CreateRemoteRegistryOption struct {
	// required: true
	Name string `json:"name" binding:"Required;MaxSize(100)"`
	// required: true
	RemoteType string `json:"remote_type" binding:"Required"`
	// required: true
	RemoteURL      string `json:"remote_url" binding:"Required;ValidUrl;MaxSize(255)"`
	RemoteUser     string `json:"remote_user" binding:"MaxSize(255)"`
	RemotePassword string `json:"remote_pass" binding:"MaxSize(255)"`
	RemoteToken    string `json:"remote_token" binding:"MaxSize(255)"`
}
