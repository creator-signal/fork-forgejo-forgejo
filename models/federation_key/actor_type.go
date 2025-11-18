// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

type ActorType int

const (
	// InvalidActorType represents an invalid ActivityPub actor type for the local database
	InvalidActorType       ActorType = 0
	invalidActorTypeString string    = ""
	// FederationHostTable represents the local database table for federation hosts.
	FederationHostType       ActorType = 1
	federationHostTypeString string    = "federation_host"
	// FederatedUserTable represents the local database table for federated users.
	FederatedUserType       ActorType = 2
	federatedUserTypeString string    = "federated_user"
)

// String gets the string representation of the ActorType
func (a ActorType) String() string {
	switch a {
	case FederationHostType:
		return federationHostTypeString
	case FederatedUserType:
		return federatedUserTypeString
	default:
		return invalidActorTypeString
	}
}

// Validate collects error strings in a slice
func (a ActorType) Validate() []string {
	switch a {
	case FederationHostType, FederatedUserType:
		return []string{}
	default:
		return []string{"invalid ActivityPub actor type"}
	}
}
