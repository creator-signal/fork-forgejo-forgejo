// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation_key

import (
	"crypto/x509"
	"fmt"

	"forgejo.org/modules/validation"
)

// FederationPublicKey data type.
type FederationPublicKey struct {
	ID    int64  `xorm:"pk autoincr"`
	KeyID KeyID  `xorm:"key_id UNIQUE NOT NULL"`
	Key   []byte `xorm:"BLOB NOT NULL"`
}

// NewFederationPublicKey creates and validates a new FederationPublicKey.
func NewFederationPublicKey(id int64, rawKeyID string, key []byte) (*FederationPublicKey, error) {
	keyID, err := NewKeyID(rawKeyID)
	if err != nil {
		return nil, err
	}

	pk := &FederationPublicKey{ID: id, KeyID: keyID, Key: key}
	_, err = validation.IsValid(pk)

	return pk, err
}

// Validate collects error strings in a slice and returns this
func (key FederationPublicKey) Validate() []string {
	var result []string

	result = append(result, key.validateKeyID()...)
	result = append(result, key.validateKey()...)

	return result
}

// validateKeyID validates that the KeyID is a non-empty, valid IRI
func (key FederationPublicKey) validateKeyID() []string {
	return key.KeyID.Validate()
}

// validateKey validates that the public key is non-empty, and properly encoded
func (key FederationPublicKey) validateKey() []string {
	result := validation.ValidateNotEmpty(string(key.Key), "Key")
	if _, err := x509.ParsePKIXPublicKey(key.Key); err != nil {
		result = append(result, fmt.Sprintf("PublicKey is not valid: %s", err))
	}
	return result
}
