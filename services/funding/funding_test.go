// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding_test

import (
	"fmt"
	"testing"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	funding_service "forgejo.org/services/funding"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestFundingConfigParse(t *testing.T) {
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
	cases := []string{
		".forgejo/FUNDING.yaml",
		".forgejo/FUNDING.yml",
		".forgejo/Funding.yaml",
		".forgejo/Funding.yml",
		".forgejo/funding.yml",
		".forgejo/funding.yaml",
		".forgejo/fundING.yml",

		".github/FUNDING.yaml",
		".github/FUNDING.yml",
		".github/Funding.yaml",
		".github/Funding.yml",
		".github/funding.yml",
		".github/funding.yaml",
		".github/fundING.yml",

		"FUNDING.yaml",
		"FUNDING.yml",
		"Funding.yaml",
		"Funding.yml",
		"funding.yml",
		"funding.yaml",
		"fundING.yml",
	}
	for _, fileName := range cases {
		t.Run(fileName, func(t *testing.T) {
			assert.True(t, funding_service.IsFundingConfig(fileName))
		})
	}
}

func TestIsNotFundingConfig(t *testing.T) {
	cases := []string{
		"README.md",
		".gitea/FUNDING.yml",
		"custom/FUNDING.yml",
		".forgejo/_FUNDING.yml",
		".forgejo/.FUNDING.yml",
		".forgejo/FUNDING.yml.",
		"",
	}
	for _, fileName := range cases {
		t.Run(fileName, func(t *testing.T) {
			assert.False(t, funding_service.IsFundingConfig(fileName))
		})
	}
}

func getFundingFromConfig(t *testing.T, config string) ([]*api.RepoFundingEntry, []error) {
	data, err := funding_service.GetFundingFromBlob([]byte(config))
	require.NoError(t, err)
	funding := data.EntryList
	errs := data.Errs
	return funding, errs
}

func assertEntry(t *testing.T, entry *api.RepoFundingEntry, expectedProvider, expectedText, expectedURL string) {
	t.Helper()
	assert.Equal(t, expectedProvider, entry.ProviderName)
	assert.Equal(t, expectedText, entry.Text)
	assert.Equal(t, expectedURL, entry.URL)
}

func assertCustom(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "custom", expectedText, expectedURL)
}

func assertLiberapay(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "liberapay", expectedText, expectedURL)
}

func assertKoFi(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "ko_fi", expectedText, expectedURL)
}

func assertPatreon(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "patreon", expectedText, expectedURL)
}

func TestConfigParseErrors(t *testing.T) {
	configs := []string{
		`this isn't yaml`,
		`[`,
		`]`,
		`foo`,
	}

	for _, config := range configs {
		data, err := funding_service.GetFundingFromBlob([]byte(config))
		require.Error(t, err)
		require.Nil(t, data)
		// there's a huge variety of possible YAML parse errors, so we won't bother testing that they're exact.
		// what's important here is that we get an error instead of data.
	}
}

func TestEntriesFromConfig(t *testing.T) {
	defer test.MockProtect(&setting.FundingProviders)()
	setting.LoadBuiltInFundingProviders()

	t.Run("Empty config", func(t *testing.T) {
		funding, errs := getFundingFromConfig(t, "")
		assert.Empty(t, errs)
		assert.Empty(t, funding)
	})

	t.Run("Custom string array", func(t *testing.T) {
		config := "custom:\n" +
			`- "https://a.com"` + "\n" +
			"- b.com\n" +
			`- "http://withquery.example.com?test=foo"` + "\n" +
			`- "http://thistimewithhash#foo"` + "\n"
		funding, errs := getFundingFromConfig(t, config)
		assert.Empty(t, errs)

		require.Len(t, funding, 4)
		assertCustom(t, funding[0], "https://a.com", "https://a.com")
		assertCustom(t, funding[1], "b.com", "http://b.com")
		assertCustom(t, funding[2], "http://withquery.example.com?test=foo", "http://withquery.example.com?test=foo")
		assertCustom(t, funding[3], "http://thistimewithhash#foo", "http://thistimewithhash#foo")
	})
}

func TestEntriesWithErrorsFromConfig(t *testing.T) {
	defer test.MockProtect(&setting.FundingProviders)()
	setting.LoadBuiltInFundingProviders()

	t.Run("Skips duplicate entries", func(t *testing.T) {
		configs := [][3]any{
			{`custom: ["https://a.com", "https://a.com", "https://b.com"]`, "https://a.com", []string{"https://a.com", "https://b.com"}},
			{`custom: [test1, test1, test2]`, "http://test1", []string{"http://test1", "http://test2"}},
			{`custom: [test1, "http://test1", test2]`, "http://test1", []string{"http://test1", "http://test2"}},
		}
		for _, configCase := range configs {
			config := configCase[0].(string)
			expectedDuplicate := configCase[1].(string)
			expectedRemainder := configCase[2].([]string)
			funding, errs := getFundingFromConfig(t, config)

			assert.Len(t, errs, 1)
			assert.Equal(t, fmt.Sprintf("Duplicate entry for key 'custom': %s", expectedDuplicate), errs[0].Error())

			actualRemainder := make([]string, 0, len(funding))
			for _, funding := range funding {
				actualRemainder = append(actualRemainder, funding.URL)
			}
			assert.Equal(t, expectedRemainder, actualRemainder)
		}
	})

	t.Run("Invalid config", func(t *testing.T) {
		configs := []string{
			`custom: [[test]]`,
			`custom: [["test"]]`,
			`custom: [[]]`,
			`custom: 42`,
			"custom: 42\nwhatever:",
		}
		for _, config := range configs {
			funding, errs := getFundingFromConfig(t, config)
			assert.Empty(t, funding)
			require.NotEmpty(t, errs)
			assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", errs[0].Error())
		}
	})

	t.Run("Partially invalid (bad key omitted)", func(t *testing.T) {
		config := "liberapay: test\n" +
			"ko_fi:\n" +
			`custom: [test, "https://example.com"]`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", errs[0].Error())

		assert.Len(t, funding, 3)
		assertLiberapay(t, funding[0], "liberapay.com/test", "https://liberapay.com/test")
		assertCustom(t, funding[1], "test", "http://test")
		assertCustom(t, funding[2], "https://example.com", "https://example.com")
	})

	t.Run("Partially invalid (unknown key omitted)", func(t *testing.T) {
		configs := []string{
			"whatever: test\ncustom: [test, \"https://example.com\"]",
			"whatever: 42\ncustom: [test, \"https://example.com\"]", // bad key type has same error
		}

		for _, config := range configs {
			funding, errs := getFundingFromConfig(t, config)

			assert.Len(t, errs, 1)
			assert.Equal(t, "Unknown funding provider: whatever", errs[0].Error())

			assert.Len(t, funding, 2)
			assertCustom(t, funding[0], "test", "http://test")
			assertCustom(t, funding[1], "https://example.com", "https://example.com")
		}
	})

	t.Run("Partially invalid (bad known keys omitted)", func(t *testing.T) {
		config := "whatever: test\n" +
			"ko_fi: 42\n" +
			`custom: [test, "https://example.com"]`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 2)
		assert.Equal(t, "Unknown funding provider: whatever", errs[0].Error())
		assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", errs[1].Error())

		assert.Len(t, funding, 2)
		assertCustom(t, funding[0], "test", "http://test")
		assertCustom(t, funding[1], "https://example.com", "https://example.com")
	})

	t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
		config := `custom: [42, "https://example.com"]`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Invalid type for key 'custom', expected a string or string array", errs[0].Error())

		assert.Len(t, funding, 1)
		assertCustom(t, funding[0], "https://example.com", "https://example.com")
	})

	t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
		config := "custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Expected up to 4 of funding provider custom", errs[0].Error())

		assert.Len(t, funding, 4)
		assertCustom(t, funding[0], "test1", "http://test1")
		assertCustom(t, funding[1], "https://example.com", "https://example.com")
		assertCustom(t, funding[2], "test3", "http://test3")
		assertCustom(t, funding[3], "test4", "http://test4")
	})

	t.Run("Partially invalid (too many of one provider, valid others)", func(t *testing.T) {
		config := "ko_fi: test\n" +
			"patreon: test\n" +
			"custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Expected up to 4 of funding provider custom", errs[0].Error())

		assert.Len(t, funding, 6)
		assertKoFi(t, funding[0], "ko-fi.com/test", "https://ko-fi.com/test")
		assertPatreon(t, funding[1], "patreon.com/test", "https://patreon.com/test")
		assertCustom(t, funding[2], "test1", "http://test1")
		assertCustom(t, funding[3], "https://example.com", "https://example.com")
		assertCustom(t, funding[4], "test3", "http://test3")
		assertCustom(t, funding[5], "test4", "http://test4")
	})

	t.Run("Partially invalid (too many of two providers, valid list of others)", func(t *testing.T) {
		config := "ko_fi: [test]\n" +
			"tidelift: npm/example\n" +
			"custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 2)
		assert.Equal(t, "Funding provider tidelift is not allowed", errs[0].Error())
		assert.Equal(t, "Expected up to 4 of funding provider custom", errs[1].Error())

		assert.Len(t, funding, 5)
		assertKoFi(t, funding[0], "ko-fi.com/test", "https://ko-fi.com/test")
		assertCustom(t, funding[1], "test1", "http://test1")
		assertCustom(t, funding[2], "https://example.com", "https://example.com")
		assertCustom(t, funding[3], "test3", "http://test3")
		assertCustom(t, funding[4], "test4", "http://test4")
	})

	t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
		config := "ko_fi: [test, test2]\n" +
			"custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 2)
		assert.Equal(t, "Expected up to 1 of funding provider ko_fi", errs[0].Error())
		assert.Equal(t, "Expected up to 4 of funding provider custom", errs[1].Error())

		assert.Len(t, funding, 5)
		assertKoFi(t, funding[0], "ko-fi.com/test", "https://ko-fi.com/test")
		assertCustom(t, funding[1], "test1", "http://test1")
		assertCustom(t, funding[2], "https://example.com", "https://example.com")
		assertCustom(t, funding[3], "test3", "http://test3")
		assertCustom(t, funding[4], "test4", "http://test4")
	})
}
