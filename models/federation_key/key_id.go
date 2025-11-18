// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"fmt"

	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// KeyID represents the public key ID used in ActivityPub federation.
type KeyID ap.IRI

// NewKeyID creates and validates an ActivityPub key ID.
func NewKeyID(id string) (KeyID, error) {
	keyID := KeyID(ap.IRI(id))
	_, err := validation.IsValid(keyID)

	return keyID, err
}

// String gets the KeyID ActivityPub IRI representation.
func (key KeyID) IRI() ap.IRI {
	return ap.IRI(key)
}

// String gets the KeyID string representation.
func (key KeyID) String() string {
	return ap.IRI(key).String()
}

// Validate collects error strings in a slice and returns this
func (key KeyID) Validate() []string {
	result := validation.ValidateNotEmpty(key.String(), "KeyID")

	uri, err := key.IRI().URL()
	if err != nil {
		result = append(result, fmt.Sprintf("invaild KeyID URL: %v", err))
	}

	if len(uri.Fragment) == 0 {
		result = append(result, "invalid KeyID fragment identifier")
	}
	if len(uri.Query()) > 0 {
		result = append(result, "invalid KeyID includes a query part")
	}
	if uri.Scheme != "http" && uri.Scheme != "https" {
		result = append(result, fmt.Sprintf("invalid KeyID scheme: %v", uri.Scheme))
	}

	return result
}
