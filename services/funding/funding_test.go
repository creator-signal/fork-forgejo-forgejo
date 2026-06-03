// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding_test

import (
	"testing"

	funding_service "forgejo.org/services/funding"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestConfigParse(t *testing.T) {
	// custom UnmarshalYAML implementation should let us gather ordering info on mapping pairs
	t.Run("no data", func(t *testing.T) {
		badConfigs := []string{
			"",
			"{}",
		}

		for _, configContent := range badConfigs {
			config := make(funding_service.RawRepoFundingConfig, 0)
			err := yaml.Unmarshal([]byte(configContent), &config)
			require.NoError(t, err)
			assert.Empty(t, config)
		}
	})

	t.Run("bad data", func(t *testing.T) {
		badConfigs := []string{
			"42",
			"[42]",
			"['42']",
			"['']",
			"[]",
			"foo:bar",
			"あ",
		}

		for _, configContent := range badConfigs {
			config := make(funding_service.RawRepoFundingConfig, 0)
			err := yaml.Unmarshal([]byte(configContent), &config)
			assert.Error(t, err, "config content: %v", configContent)
		}
	})

	t.Run("dummy single data", func(t *testing.T) {
		cases := [][2]any{
			{"foo: bar", funding_service.RawRepoFundingConfigEntry{"foo", "bar"}},
			{"foo: [bar]", funding_service.RawRepoFundingConfigEntry{"foo", []any{"bar"}}},
			{"foo: 42", funding_service.RawRepoFundingConfigEntry{"foo", 42}},
			{"foo: [42]", funding_service.RawRepoFundingConfigEntry{"foo", []any{42}}},
			{"foo:", funding_service.RawRepoFundingConfigEntry{"foo", nil}},
			{"あ: '!'", funding_service.RawRepoFundingConfigEntry{"あ", "!"}},
			{"あ:", funding_service.RawRepoFundingConfigEntry{"あ", nil}},
		}

		for _, c := range cases {
			configContent := c[0].(string)
			expected := c[1].(funding_service.RawRepoFundingConfigEntry)
			config := make(funding_service.RawRepoFundingConfig, 0)
			err := yaml.Unmarshal([]byte(configContent), &config)
			require.NoError(t, err)
			assert.Len(t, config, 1)
			assert.Equal(t, expected.Key, config[0].Key)
			assert.Equal(t, expected.Value, config[0].Value)
		}
	})

	t.Run("duplicate keys", func(t *testing.T) {
		configContent := "a: b\nfoo: bar\nfoo: baz"
		config := make(funding_service.RawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.Error(t, err)
		assert.Equal(t, "Duplicate YAML key: foo", err.Error())
	})

	t.Run("orders a then b then c", func(t *testing.T) {
		configContent := "a: a\nb: b\nc: c"
		config := make(funding_service.RawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, "a", config[0].Key)
		assert.Equal(t, "a", config[0].Value)
		assert.Equal(t, "b", config[1].Key)
		assert.Equal(t, "b", config[1].Value)
		assert.Equal(t, "c", config[2].Key)
		assert.Equal(t, "c", config[2].Value)
	})

	t.Run("orders b then a then c", func(t *testing.T) {
		configContent := "b: b\na: a\nc: c"
		config := make(funding_service.RawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, "b", config[0].Key)
		assert.Equal(t, "b", config[0].Value)
		assert.Equal(t, "a", config[1].Key)
		assert.Equal(t, "a", config[1].Value)
		assert.Equal(t, "c", config[2].Key)
		assert.Equal(t, "c", config[2].Value)
	})

	t.Run("orders c then a then b", func(t *testing.T) {
		configContent := "c: c\na: a\nb: b"
		config := make(funding_service.RawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, "c", config[0].Key)
		assert.Equal(t, "c", config[0].Value)
		assert.Equal(t, "a", config[1].Key)
		assert.Equal(t, "a", config[1].Value)
		assert.Equal(t, "b", config[2].Key)
		assert.Equal(t, "b", config[2].Value)
	})

	t.Run("orders c then b then a", func(t *testing.T) {
		configContent := "c: c\nb: b\na: a"
		config := make(funding_service.RawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, "c", config[0].Key)
		assert.Equal(t, "c", config[0].Value)
		assert.Equal(t, "b", config[1].Key)
		assert.Equal(t, "b", config[1].Value)
		assert.Equal(t, "a", config[2].Key)
		assert.Equal(t, "a", config[2].Value)
	})
}

func TestIsFundingConfig(t *testing.T) {
	assert.True(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".forgejo/fundING.yml"))

	assert.True(t, funding_service.IsFundingConfig(".github/FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/funding.yml"))
	assert.True(t, funding_service.IsFundingConfig(".github/funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig(".github/fundING.yml"))

	assert.True(t, funding_service.IsFundingConfig("FUNDING.yaml"))
	assert.True(t, funding_service.IsFundingConfig("FUNDING.yml"))
	assert.True(t, funding_service.IsFundingConfig("Funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig("Funding.yml"))
	assert.True(t, funding_service.IsFundingConfig("funding.yml"))
	assert.True(t, funding_service.IsFundingConfig("funding.yaml"))
	assert.True(t, funding_service.IsFundingConfig("fundING.yml"))

	assert.False(t, funding_service.IsFundingConfig("README.md"))
	assert.False(t, funding_service.IsFundingConfig(".gitea/FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig("custom/FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/_FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/.FUNDING.yml"))
	assert.False(t, funding_service.IsFundingConfig(".forgejo/FUNDING.yml."))
}
