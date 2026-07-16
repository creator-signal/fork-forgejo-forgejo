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

func TestFundingProviderConfigCleanUpSigils(t *testing.T) {
	// if the result expects more than one input, or an incomplete formatting sigil, then fmt.Sprintf may complain!
	// we don't worry here about strings that take zero inputs, we don't know where one should go!
	cases := [][2]string{
		{"", ""},
		{"%", "%%"},
		{"a", "a"},
		{"%s", "%[1]s"},
		{"%%s", "%%s"},
		{"%s%s", "%[1]s%[1]s"},
		{"%s%%s", "%[1]s%%s"},
		{"%%s%s", "%%s%[1]s"},
		{"%[1]s", "%[1]s"},
		{"%[1s", "%%[1s"},
		{"%1s", "%%1s"},
		{"%1]s", "%%1]s"},
		{"%[2]s", "%%[2]s"},
		{"%[14]s", "%%[14]s"},
		{"%よ", "%%よ"},
		{"%よ%", "%%よ%%"},
		{"%yo", "%%yo"},
		{"%yo%", "%%yo%%"},
		{"http://example.com/%%s", "http://example.com/%%s"},
		{"http://example.com/%%%%s", "http://example.com/%%%%s"},
		{"http://example.com/%%%[1]s", "http://example.com/%%%[1]s"},
		{"http://example.com/%%[1]s", "http://example.com/%%[1]s"},
		{"https://example.com/%s?u=%s", "https://example.com/%[1]s?u=%[1]s"},
		{"https://example.com/%[1]s", "https://example.com/%[1]s"},
		{"https://example.com/%[1]s?q=%[1]s", "https://example.com/%[1]s?q=%[1]s"},
		{"https://example.com/%[2]s", "https://example.com/%%[2]s"},
		{"https://example.com/%[2]s/%s", "https://example.com/%%[2]s/%[1]s"},
		{"https://example.com/%[2]s/%[1]s", "https://example.com/%%[2]s/%[1]s"},
		{"https://example.com/%[2]s/%[1]s/%s", "https://example.com/%%[2]s/%[1]s/%[1]s"},
	}

	for _, c := range cases {
		input := c[0]
		expected := c[1]
		assert.Equal(t, expected, cleanUpSigils(input))
	}
}

func TestFundingProviderConfigIgnoresInvalidTemplate(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	// a URL template string that doesn't take exactly one input may cause errors.
	// the cleanUpSigils function handles extra sigils, but may result in strings that take zero inputs, which are invalid here!
	cases := []string{
		"",
		"s",
		"https://example.com",
		"%[2]s",
	}

	for _, url := range cases {
		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
URL = "%s"

[funding.mycustom2]
URL = "%%s"

[funding.nothing]
`, url))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)
		assert.Nil(t, FundingProviders["mycustom"])
		assert.Nil(t, FundingProviders["nothing"])
		assert.NotNil(t, FundingProviders["mycustom2"])
		assert.Equal(t, 15, MaxFundingEntriesPerConfig) // default value
	}
}

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

func TestFundingProviderConfigIgnoredOverride(t *testing.T) {
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

func TestFundingProviderConfigMaxEntries(t *testing.T) {
	defer test.MockProtect(&MaxFundingEntriesPerConfig)()

	cases := [][2]int{
		{-1, 0}, // clamps to 0
		{0, 0},
		{1, 1},
		{15, 15},
		{16, 16},
		{20, 20},
		{21, 20}, // clamps to 20
	}

	for _, c := range cases {
		limit := c[0]
		expected := c[1]
		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding]
MAX_FUNDING_ENTRIES_PER_CONFIG = %d
`, limit))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		assert.Equal(t, expected, MaxFundingEntriesPerConfig)
	}
}
