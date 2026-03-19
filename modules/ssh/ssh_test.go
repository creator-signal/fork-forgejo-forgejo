// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package ssh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCertificatePrincipalMatching(t *testing.T) {
	cases := []struct {
		Allowed  string
		Supplied string
		Matched  bool
	}{
		{"", "", false},
		{"", "username", true},
		{"", "garbage", false},
		{"", "username,garbage", true},
		{"", "foo,bar", false},

		{"username", "", false},
		{"username", "username", true},
		{"username", "garbage", false},
		{"username", "username,garbage", true},
		{"username", "foo,bar", false},

		{"garbage", "", false},
		{"garbage", "username", false},
		{"garbage", "garbage", true},
		{"garbage", "username,garbage", true},
		{"garbage", "foo,bar", false},

		{"username,garbage", "", false},
		{"username,garbage", "username", true},
		{"username,garbage", "garbage", true},
		{"username,garbage", "username,garbage", true},
		{"username,garbage", "foo,bar", false},

		{"username", "USERNAME", false},

		{"username,garbage", "erna", false},
		{"username,garbage", "user", false},
		{"username,garbage", "name", false},
		{"username,garbage", "bage", false},

		{"username,username", "username", true},
		{"username", "username,username", true},
		{"a,b,c", "c", true},
		{"a,c,b", "c", true},
		{"c,a,b", "c", true},
		{"a,d", "a,b,c", true},
		{"d,b", "a,b,c", true},
		{"v,w", "x,y,z", false},
	}

	const defaultUsername = "username"

	for _, c := range cases {
		allowed := splitPrincipals(c.Allowed)
		supplied := splitPrincipals(c.Supplied)
		if len(allowed) == 0 {
			allowed = []string{defaultUsername}
		}
		_, found := findMatchingPrincipal(supplied, allowed)
		assert.Equal(t, c.Matched, found, "Expected that for allowed %q and supplied %q, found should be %v", c.Allowed, c.Supplied, c.Matched)
	}
}
