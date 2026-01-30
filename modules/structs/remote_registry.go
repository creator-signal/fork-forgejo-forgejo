// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// RemoteRegistry the response when a remote registry gets successfully created
// swagger:model
type RemoteRegistry struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	OwnerType  string `json:"owner_type"`
	OwnerID    int64  `json:"owner_id"`
	RemoteURL  string `json:"remote_url"`
	RemoteHost string `json:"remote_host"`
	RemotePort uint16 `json:"remote_port"`
	RemoteUser string `json:"remote_user"`
	RemoteType string `json:"remote_type"`
}

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
	TestConnection bool   `json:"test_connection"`
}
