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

func TestNewFundingProviderConfig(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	// only requires url formatter. the text, limits, and icon can be derived automatically
	cfg, err := NewConfigProviderFromData(`
[funding.mycustom]
URL = "https://mycustom.example.com/%s"
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	mycustom := FundingProviders["mycustom"]
	assert.Equal(t, "mycustom", mycustom.Name)
	assert.Equal(t, "mycustom.svg", mycustom.IconName)
	assert.Equal(t, 1, int(mycustom.Limit))
	assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text) // derived from URL
	assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL) // note that the %s becomes %[1]s as well, since this will only ever have the one input
}

func TestNewFundingProviderConfigWithMailtoUrl(t *testing.T) {
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
	assert.Equal(t, "mycustom.svg", mycustom.IconName)
	assert.Equal(t, 1, int(mycustom.Limit))
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.Text) // same as URL
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.URL)
}

func TestNewFundingProviderConfigWithMailtoUrlWithText(t *testing.T) {
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
	assert.Equal(t, "mycustom.svg", mycustom.IconName)
	assert.Equal(t, 1, int(mycustom.Limit))
	assert.Equal(t, "Email %[1]s@localhost for info", mycustom.Text)
	assert.Equal(t, "mailto:%[1]s@localhost", mycustom.URL)
}

func TestNewFundingProviderConfigWithText(t *testing.T) {
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
	assert.Equal(t, "mycustom.svg", mycustom.IconName)
	assert.Equal(t, 1, int(mycustom.Limit))
	assert.Equal(t, "mycustom.example.lol/%[1]s", mycustom.Text)
	assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL)
}

func TestNewFundingProviderConfigHandlesSigils(t *testing.T) {
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
		expectedUrl := c[2]

		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
URL = "%s"
`, url))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		mycustom := FundingProviders["mycustom"]
		assert.Equal(t, expectedText, mycustom.Text)
		assert.Equal(t, expectedUrl, mycustom.URL)
	}
}

func TestNewFundingProviderConfigWithLimit(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cases := [][2]int{
		// TODO: try floats too
		{-1, 0}, // lower bound clamps to 0
		{-12, 0},
		{-0, 0},
		{0, 0},
		{1, 1},
		{4, 4},
		{15, 15},
		{16, 16},
		{50, 16}, // upper bound clamps to 16
	}

	for _, c := range cases {
		input := c[0]
		expected := c[1]
		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
LIMIT = %d
URL = "https://mycustom.example.com/%%s"
`, input))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		mycustom := FundingProviders["mycustom"]
		assert.Equal(t, "mycustom", mycustom.Name)
		assert.Equal(t, "mycustom.svg", mycustom.IconName)
		assert.Equal(t, expected, int(mycustom.Limit))
		assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)
		assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL)
	}
}

func TestNewFundingProviderConfigWithCustomIcon(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cases := [][2]string{
		{"", "mycustom.svg"},
		{"mycustom.svg", "mycustom.svg"},
		{"mycustom.png", "mycustom.png"},
		{"img/funding/mycustom.png", "mycustom.svg"}, // defaults when not a filename
		{"any/path/here/mycustom.png", "mycustom.svg"},
		{"any/path/here/mycustom.svg", "mycustom.svg"},
		{"../mycustom.png", "mycustom.svg"},
		{"../mycustom.svg", "mycustom.svg"},
		{"./mycustom.png", "mycustom.svg"},
		{"/mycustom.png", "mycustom.svg"},
		{"\\mycustom.png", "mycustom.svg"},
	}

	for _, c := range cases {
		input := c[0]
		expected := c[1]
		cfg, err := NewConfigProviderFromData(fmt.Sprintf(`
[funding.mycustom]
URL = "https://mycustom.example.com/%%s"
ICON = "%s"
`, input))
		require.NoError(t, err)
		loadCustomFundingProvidersFrom(cfg)

		mycustom := FundingProviders["mycustom"]
		assert.Equal(t, "mycustom", mycustom.Name)
		assert.Equal(t, expected, mycustom.IconName)
		assert.Equal(t, 1, int(mycustom.Limit))
		assert.Equal(t, "mycustom.example.com/%[1]s", mycustom.Text)
		assert.Equal(t, "https://mycustom.example.com/%[1]s", mycustom.URL)
	}
}

func TestIgnoredOverrideFundingProviderConfig(t *testing.T) {
	defer test.MockProtect(&FundingProviders)()

	cfg, err := NewConfigProviderFromData(`
[funding.ko_fi]
LIMIT = 0
`)
	require.NoError(t, err)
	loadCustomFundingProvidersFrom(cfg)

	ko_fi := FundingProviders["ko_fi"]
	assert.Equal(t, "ko_fi", ko_fi.Name)
	assert.Equal(t, "ko_fi.svg", ko_fi.IconName)
	assert.Equal(t, 1, int(ko_fi.Limit)) // no change from builtin
	assert.Equal(t, "ko-fi.com/%[1]s", ko_fi.Text)
	assert.Equal(t, "https://ko-fi.com/%[1]s", ko_fi.URL)
}
