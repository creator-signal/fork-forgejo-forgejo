// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package asymkey

import (
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCAOptions(t *testing.T) {
	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, false)()

	{
		_, _, err := ParseCAOptions([]string{`cert-authority`})
		assert.True(t, IsErrSSHCADisabled(err))
	}

	defer test.MockVariableValue(&setting.SSH.EnableCertAuth, true)()

	cases := []struct {
		desc       string
		options    []string
		ok, isCA   bool
		principals string
	}{
		{"no options", nil, true, false, ""},
		{"cert-authority only", []string{`cert-authority`}, true, true, ""},
		{"cert-authority with single explicit principal", []string{`cert-authority`, `principals="x"`}, true, true, "x"},
		{"cert-authority with multiple explicit principals", []string{`cert-authority`, `principals="x,y,z"`}, true, true, "x,y,z"},
		{"cert-authority with empty explicit principals string", []string{`cert-authority`, `principals=""`}, false, false, ""},

		// See ParseCAOptions implementation and comment for the things we reject.
		// We reject MORE than OpenSSH rejects, in order to not have to implement OpenSSH's parsing logic, weirdness for weirdness.
		// If we want to allow exotic principal strings we can do so later.
		{"principals with backslash", []string{`cert-authority`, `principals="m\n"`}, false, false, ""},
		{"principals with backslash-doublequote", []string{`cert-authority`, `principals="a\"b"`}, false, false, ""},
		{"principals with spaces", []string{`cert-authority`, `principals="a, b, c"`}, false, false, ""},
		{"principals with control codes other than space (newline)", []string{`cert-authority`, "principals=\"\n\""}, false, false, ""},
		{"principals with control codes other than space (nul)", []string{`cert-authority`, "principals=\"\000\""}, false, false, ""},
		{"principals with control codes other than space (bel)", []string{`cert-authority`, "principals=\"\a\""}, false, false, ""},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			isCA, principals, err := ParseCAOptions(c.options)
			if c.ok {
				require.NoError(t, err)
				assert.Equal(t, c.isCA, isCA)
				assert.Equal(t, c.principals, principals)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
