// Copyright 2015 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// PublicKey publickey is a user key to push code to repository
type PublicKey struct {
	ID          int64  `json:"id"`
	Key         string `json:"key"`
	URL         string `json:"url,omitempty"`
	Title       string `json:"title,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	// swagger:strfmt date-time
	Created  time.Time `json:"created_at,omitempty"`
	Owner    *User     `json:"user,omitempty"`
	ReadOnly bool      `json:"read_only,omitempty"`
	KeyType  string    `json:"key_type,omitempty"`
	// TODO: I'm not sure how best to handle user-supplied SSH CA keys here.
	// In the asymkey model there are fields IsCA, Principals. We could add
	// optional fields for those here, too (in variously ugly ways), or we
	// could add `cert-authority,principals="..."` options to the Key field.
	// Which is nicer/least bad? The latter avoids confusion for a plain key
	// in code not aware of the IsCA field, but the former allows consumers to
	// avoid parsing the weird authorized_keys format. Maybe both then???
	// Raises the possibility of inconsistency! So I'm still not sure.
	// See services/convert/convert.go
}
