// Copyright 2025, 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package federation_key

import (
	"crypto/ed25519"
	"crypto/rsa"
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
	pk, err := x509.ParsePKIXPublicKey(key.Key)
	if err != nil {
		result = append(result, fmt.Sprintf("PublicKey is not valid: %s", err))
	}

	switch key.Algorithm {
	case RsaSha256Cavage, RsaSha512Cavage:
		result = append(result, key.validateRSAKey(pk)...)
	case ED25519:
		result = append(result, key.validateEd25519Key(pk)...)
	}

	return result
}

func (key FederationPublicKey) validateRSAKey(publicKey any) []string {
	pk, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return []string{"invalid RSA public key"}
	}

	size := pk.Size()

	if (size * 8) < key.Algorithm.MinKeyLength() {
		return []string{fmt.Sprintf("invalid RSA key length: %d", size*8)}
	}

	return nil
}

func (key FederationPublicKey) validateEd25519Key(publicKey any) []string {
	_, ok := publicKey.(ed25519.PublicKey)
	if !ok {
		return []string{"invalid Ed25519 public key"}
	}
	return nil
}

// validateActorID validates that the actor ID is a non-negative actor database ID
func (key FederationPublicKey) validateActorID() []string {
	var result []string
	if key.ActorID < 0 {
		result = []string{fmt.Sprintf("invalid actor ID: %v", key.ActorID)}
	}
	return result
}
