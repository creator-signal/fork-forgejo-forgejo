// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"fmt"
	"strings"

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
		// return early, don't attempt to access an invalid URL
		return append(result, fmt.Sprintf("invaild KeyID URL: %v", err))
	}
	if uri.Scheme != "http" && uri.Scheme != "https" {
		result = append(result, fmt.Sprintf("invalid KeyID scheme: %v", uri.Scheme))
	}
	if uri.User != nil && len(uri.User.String()) > 0 {
		result = append(result, "KeyID insecure user credentials, see `CWE-922: Insecure Storage of Sensitive Information`")
	}
	if strings.Contains(uri.Path, "..") {
		result = append(result, "KeyID contains relative path, possible `CWE 35: Path Traversal`")
	}
	if len(uri.Fragment) == 0 && len(uri.Query()) == 0 {
		result = append(result, "missing KeyID identifier")
	}

	return result
}
