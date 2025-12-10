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
	ID        int64     `xorm:"pk autoincr"`
	KeyID     KeyID     `xorm:"key_id UNIQUE NOT NULL"`
	Key       []byte    `xorm:"BLOB NOT NULL"`
	ActorID   int64     `xorm:"NOT NULL"`
	ActorType ActorType `xorm:"NOT NULL"`
	Algorithm Algorithm `xorm:"NOT NULL"`
}

// NewFederationPublicKey creates and validates a new FederationPublicKey.
func NewFederationPublicKey(id int64, rawKeyID string, key []byte, actorID int64, actorType ActorType, alg Algorithm) (*FederationPublicKey, error) {
	keyID, err := NewKeyID(rawKeyID)
	if err != nil {
		return nil, err
	}

	pk := &FederationPublicKey{ID: id, KeyID: keyID, Key: key, ActorID: actorID, ActorType: actorType, Algorithm: alg}
	_, err = validation.IsValid(pk)

	return pk, err
}

// Validate collects error strings in a slice and returns this
func (key FederationPublicKey) Validate() []string {
	var result []string

	result = append(result, key.validateID()...)
	result = append(result, key.validateKeyID()...)
	result = append(result, key.validateKey()...)
	result = append(result, key.validateActorID()...)
	result = append(result, key.ActorType.Validate()...)
	result = append(result, key.Algorithm.Validate()...)

	return result
}

// validateID validates that the FederationPublicKey ID is a non-negative database ID
func (key FederationPublicKey) validateID() []string {
	var result []string
	if key.ID < 0 {
		result = []string{fmt.Sprintf("invalid federation public key ID: %v", key.ID)}
	}
	return result
}

// validateKeyID validates that the KeyID is a non-empty, valid IRI
func (key FederationPublicKey) validateKeyID() []string {
	return key.KeyID.Validate()
}

// validateKey validates that the public key is non-empty, and properly encoded
func (key FederationPublicKey) validateKey() []string {
	result := validation.ValidateNotEmpty(key.Key, "Key")
	if _, err := x509.ParsePKIXPublicKey(key.Key); err != nil {
		result = append(result, fmt.Sprintf("PublicKey is not valid: %s", err))
	}
	return result
}

// validateActorID validates that the actor ID is a non-negative actor database ID
func (key FederationPublicKey) validateActorID() []string {
	var result []string
	if key.ActorID < 0 {
		result = []string{fmt.Sprintf("invalid actor ID: %v", key.ActorID)}
	}
	return result
}
