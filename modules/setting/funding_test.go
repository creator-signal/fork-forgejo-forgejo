// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package setting

import (
	"fmt"
	"testing"

	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFundingProviderConfig(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	// only requires url formatter. the text and icon can be derived automatically
	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "https://mycustom.example.com/%s"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)        // derived from URL
	assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL) // note that the %s becomes %[1]s as well, since this will only ever have the one input
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigWithMailtoUrl(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	// not sure what an email domain would do for funding stuffs, but weird formatting cases are good to test
	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "mailto:%s@localhost"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.Text) // same as URL
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.URL)
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigWithMailtoUrlWithText(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "mailto:%s@localhost"
TEXT = "Email %s@localhost for info"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "Email %[1]s@localhost for info", mycustom.Text)
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.URL)
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigWithText(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
TEXT = "mycustom.example.lol/%s" # different from url
URL = "https://mycustom.example.com/%s"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mycustom.example.lol/%[1]s", mycustom.Text)
	assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL)
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigWithSchemalessUrl(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "mycustom.example.com/%s"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)
	assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.URL)
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigWithWeirdSchemalessUrl(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "://mycustom.example.com/%s"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)
	assert.Equal(t, "://mycustom.example.com/%[1]s", mycustom.URL)
	assert.Equal(t, `^[^/]+$`, mycustom.InputPattern.String())
}

func TestFundingProviderConfigHandlesSigils(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cases := [][3]string{
		{"https://mycustom.example.com/%s?u=%s", "mycustom.example.com/%[1]s?u=%[1]s", "https://mycustom.example.com/%[1]s?u=%[1]s"},
		{"https://mycustom.example.com/%[1]s", "mycustom.example.com/%[1]s", "https://mycustom.example.com/%[1]s"},
		{"https://mycustom.example.com/%[1]s?q=%[1]s", "mycustom.example.com/%[1]s?q=%[1]s", "https://mycustom.example.com/%[1]s?q=%[1]s"},
		{"https://mycustom.example.com/%[2]s", "mycustom.example.com/%%[2]s", "https://mycustom.example.com/%%[2]s"},
		{"https://mycustom.example.com/%[2]s/%s", "mycustom.example.com/%%[2]s/%[1]s", "https://mycustom.example.com/%%[2]s/%[1]s"},
		{"https://mycustom.example.com/%[2]s/%[1]s", "mycustom.example.com/%%[2]s/%[1]s", "https://mycustom.example.com/%%[2]s/%[1]s"},
		{"https://mycustom.example.com/%[2]s/%[1]s/%s", "mycustom.example.com/%%[2]s/%[1]s/%[1]s", "https://mycustom.example.com/%%[2]s/%[1]s/%[1]s"},
		{"%s", "%[1]s", "%[1]s"},
		{"%[1]s", "%[1]s", "%[1]s"},
	}

	for _, c := range cases {
		url := c[0]
		expectedText := c[1]
		expectedURL := c[2]

		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
URL = "%s"
`, url))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		mycustom := FundingProviders["mycustom"]
		assert.Equal(t, expectedText, mycustom.Text)
		assert.Equal(t, expectedURL, mycustom.URL)
	}
}

func TestFundingProviderConfigWithCustomInputPattern(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cases := [][2]string{
		{`^[^/]+$`, `^[^/]+$`},
		{"", `^[^/]+$`}, // default value matches a single path segment
		{".+", `.+`},
		{"/.+/", `/.+/`}, // slash delimiters are taken literally (probs shouldn't use them)
		{`/^[^/]+$/`, `/^[^/]+$/`},
		{"this is kindof like a regex", `this is kindof like a regex`},
		{"this is [not a regex", `^[^/]+$`},
	}

	for _, c := range cases {
		input := c[0]
		expected := c[1]
		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
URL = "https://mycustom.example.com/%%s"
INPUT_PATTERN = %v
`, input))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		mycustom := FundingProviders["mycustom"]
		assert.Equal(t, "mycustom", mycustom.Name)
		assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)
		assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL)
		assert.Equal(t, expected, mycustom.InputPattern.String())
	}
}

func TestIgnoredOverrideFundingProviderConfig(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.ko_fi]
URL = example.com/%[1]s
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	koFi := FundingProviders["ko_fi"]
	assert.Equal(t, "ko_fi", koFi.Name)
	assert.Equal(t, "ko-fi.com/%[1]s", koFi.Text) // no change from builtin
	assert.Equal(t, "https://ko-fi.com/%[1]s", koFi.URL)
	assert.Equal(t, `^[^/]+$`, koFi.InputPattern.String())
}
