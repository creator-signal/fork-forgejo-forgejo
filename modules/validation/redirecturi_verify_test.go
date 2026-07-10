// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_IsValidOAuthRedirectURI(t *testing.T) {
	cases := []struct {
		description string
		uri         string
		valid       bool
	}{
		{"Custom application scheme", "x-app-auth://oauth", true},
		{"Opaque private-use scheme", "com.example.app:/oauth2redirect", true},
		{"HTTPS callback", "https://example.com/callback", true},
		{"Loopback HTTP callback", "http://127.0.0.1:8080/callback", true},
		{"Empty string", "", false},
		{"Whitespace only", "   ", false},
		{"Missing colon after scheme", "http//example.com/cb", false},
		{"Relative path without scheme", "/relative/callback", false},
		{"Host and path without scheme", "example.com/callback", false},
		{"Fragment component", "https://example.com/cb#frag", false},
		{"Invalid port", "http://example.com:99x/", false},
		{"javascript pseudo-scheme", "javascript:alert(document.domain)", false},
		{"data pseudo-scheme", "data:text/html,<script>alert(1)</script>", false},
	}

	for _, testCase := range cases {
		t.Run(testCase.description, func(t *testing.T) {
			assert.Equal(t, testCase.valid, IsValidOAuthRedirectURI(testCase.uri))
		})
	}
}
