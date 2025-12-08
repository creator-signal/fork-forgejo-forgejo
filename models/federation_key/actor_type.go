// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"fmt"
)

type ActorType string

const (
	// FederationHostTable represents the local database table for federation hosts.
	FederationHostType ActorType = "federation_host"
	// FederatedUserTable represents the local database table for federated users.
	FederatedUserType ActorType = "federated_user"
)

// Validate collects error strings in a slice
func (a ActorType) Validate() []string {
	switch a {
	case FederationHostType, FederatedUserType:
		return []string{}
	default:
		return []string{fmt.Sprintf("invalid ActivityPub actor type: %v", a)}
	}
}
