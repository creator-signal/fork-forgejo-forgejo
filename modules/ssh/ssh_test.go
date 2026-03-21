// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package ssh

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type PrincipalMatchingTestCase = struct {
	Allowed  string
	Supplied string
	Matched  bool
}

// Exposed for use in E2E tests, where we need the same kind of list
func VariousPrincipalMatchingTestCases(userName string, otherUser string) []PrincipalMatchingTestCase {
	userAndOther := userName + "," + otherUser
	unrelated := userName + "_," + otherUser + "_"
	repeated := userName + "," + userName

	return []PrincipalMatchingTestCase{
		{"", "", false},
		{"", userName, true},
		{"", otherUser, false},
		{"", userAndOther, true},
		{"", unrelated, false},

		{userName, "", false},
		{userName, userName, true},
		{userName, otherUser, false},
		{userName, userAndOther, true},
		{userName, unrelated, false},

		{otherUser, "", false},
		{otherUser, userName, false},
		{otherUser, otherUser, true},
		{otherUser, userAndOther, true},
		{otherUser, unrelated, false},

		{userAndOther, "", false},
		{userAndOther, userName, true},
		{userAndOther, otherUser, true},
		{userAndOther, userAndOther, true},
		{userAndOther, unrelated, false},

		{userName, strings.ToUpper(userName), false},

		{userAndOther, userName[1 : len(userName)-1], false},
		{userAndOther, userName[:len(userName)/2], false},
		{userAndOther, userName[len(userName)/2:], false},
		{userAndOther, otherUser[len(otherUser)/2:], false},

		{repeated, userName, true},
		{userName, repeated, true},
	}
}

func TestCertificatePrincipalMatching(t *testing.T) {
	userName := "username"

	for _, c := range append(VariousPrincipalMatchingTestCases(userName, "garbage"), []PrincipalMatchingTestCase{
		{"a,b,c", "c", true},
		{"a,c,b", "c", true},
		{"c,a,b", "c", true},
		{"a,d", "a,b,c", true},
		{"d,b", "a,b,c", true},
		{"v,w", "x,y,z", false},
	}...) {
		allowed := splitPrincipals(c.Allowed)
		supplied := splitPrincipals(c.Supplied)
		if len(allowed) == 0 {
			allowed = []string{userName}
		}
		_, found := findMatchingPrincipal(supplied, allowed)
		assert.Equal(t, c.Matched, found, "Expected that for allowed %q and supplied %q, found should be %v", c.Allowed, c.Supplied, c.Matched)
	}
}
