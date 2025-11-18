// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package util

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
)

// GenerateKeyPair generates a public and private keypair
func GenerateKeyPair(bits int) (string, string, error) {
	priv, _ := rsa.GenerateKey(rand.Reader, bits)
	privPem := pemBlockForPriv(priv)
	pubPem, err := pemBlockForPub(&priv.PublicKey)
	if err != nil {
		return "", "", err
	}
	return privPem, pubPem, nil
}

func pemBlockForPriv(priv *rsa.PrivateKey) string {
	privBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	return string(privBytes)
}

func pemBlockForPub(pub *rsa.PublicKey) (string, error) {
	pubASN1, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", err
	}
	pubBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubASN1,
	})
	return string(pubBytes), nil
}

// RandomPKIXPublicKeyx509 generates a random RSA keypair.
//
// `bits` is the desired bit-length for the keypair.
//
// Returns the x509 asn1 DER-encoded public key bytes.
func RandomPKIXPublicKey(bits int) ([]byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, err
	}
	return x509.MarshalPKIXPublicKey(priv.Public())
}

// CreatePublicKeyFingerprint creates a fingerprint of the given key.
// The fingerprint is the sha256 sum of the PKIX structure of the key.
func CreatePublicKeyFingerprint(key crypto.PublicKey) ([]byte, error) {
	bytes, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return nil, err
	}

	checksum := sha256.Sum256(bytes)

	return checksum[:], nil
}
