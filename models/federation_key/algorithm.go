// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

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
