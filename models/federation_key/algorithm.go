// Copyright 2025, 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package federation_key

import (
	"fmt"

	"github.com/42wim/httpsig"
)

type Algorithm string

const (
	RsaSha256Cavage Algorithm = Algorithm(string(httpsig.RSA_SHA256))
	RsaSha512Cavage Algorithm = Algorithm(string(httpsig.RSA_SHA512))
	ED25519         Algorithm = Algorithm(string(httpsig.ED25519))
)

// Validate collects error strings in a slice
func (a Algorithm) Validate() []string {
	switch a {
	case RsaSha256Cavage, RsaSha512Cavage, ED25519:
		return []string{}
	default:
		return []string{fmt.Sprintf("invalid ActivityPub signature algorithm: %v", a)}
	}
}

// MinKeyLength returns the minimum key length (in bits) for the public key.
func (a Algorithm) MinKeyLength() int {
	switch a {
	case RsaSha256Cavage, RsaSha512Cavage:
		return 2048
	case ED25519:
		return 256
	default:
		return 0
	}
}
