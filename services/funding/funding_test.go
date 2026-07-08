// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package funding

import (
	"fmt"
	"testing"

	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"

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
			config := make(rawRepoFundingConfig, 0)
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
			config := make(rawRepoFundingConfig, 0)
			err := yaml.Unmarshal([]byte(configContent), &config)
			assert.Error(t, err, "config content: %v", configContent)
		}
	})

	t.Run("dummy single data", func(t *testing.T) {
		cases := [][2]any{
			{"foo: bar", rawRepoFundingConfigEntry{"foo", "bar"}},
			{"foo: [bar]", rawRepoFundingConfigEntry{"foo", []any{"bar"}}},
			{"foo: 42", rawRepoFundingConfigEntry{"foo", 42}},
			{"foo: [42]", rawRepoFundingConfigEntry{"foo", []any{42}}},
			{"foo:", rawRepoFundingConfigEntry{"foo", nil}},
			{"foo: 'localhost:4242'", rawRepoFundingConfigEntry{"foo", "localhost:4242"}},
			{"foo: 'localhost: 4242'", rawRepoFundingConfigEntry{"foo", "localhost: 4242"}},
			{"あ: '!'", rawRepoFundingConfigEntry{"あ", "!"}},
			{"あ:", rawRepoFundingConfigEntry{"あ", nil}},
			{`custom: '😀.com'`, rawRepoFundingConfigEntry{"custom", "😀.com"}},
			// adapted from the list at https://en.wikipedia.org/wiki/International_email:
			{"bar: '用户@例子.广告'", rawRepoFundingConfigEntry{"bar", "用户@例子.广告"}}, // at char is valid
			{"bar: 'ಡೇಟಾಮೇಲ್.ಭಾರತ'", rawRepoFundingConfigEntry{"bar", "ಡೇಟಾಮೇಲ್.ಭಾರತ"}},
			{"bar: 'डाटा.भारत'", rawRepoFundingConfigEntry{"bar", "डाटा.भारत"}},
			{"bar: 'пошта.укр'", rawRepoFundingConfigEntry{"bar", "пошта.укр"}},
			{"bar: 'παράδειγμα.ελ'", rawRepoFundingConfigEntry{"bar", "παράδειγμα.ελ"}},
			{"bar: 'Sörensen.example.com'", rawRepoFundingConfigEntry{"bar", "Sörensen.example.com"}},
			{"bar: 'пример.рф'", rawRepoFundingConfigEntry{"bar", "пример.рф"}},
			{"bar: 'موقع.عر'", rawRepoFundingConfigEntry{"bar", "موقع.عر"}},
		}

		for _, c := range cases {
			configContent := c[0].(string)
			expected := c[1].(rawRepoFundingConfigEntry)
			config := make(rawRepoFundingConfig, 0)
			err := yaml.Unmarshal([]byte(configContent), &config)
			require.NoError(t, err)
			assert.Len(t, config, 1)
			assert.Equal(t, expected, config[0])
		}
	})

	t.Run("duplicate keys", func(t *testing.T) {
		configContent := "a: b\nfoo: bar\nfoo: baz"
		config := make(rawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.Error(t, err)
		assert.Equal(t, "Duplicate YAML key: foo", err.Error())
	})

	t.Run("orders a then b then c", func(t *testing.T) {
		configContent := "a: a\nb: b\nc: c"
		config := make(rawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, rawRepoFundingConfigEntry{"a", "a"}, config[0])
		assert.Equal(t, rawRepoFundingConfigEntry{"b", "b"}, config[1])
		assert.Equal(t, rawRepoFundingConfigEntry{"c", "c"}, config[2])
	})

	t.Run("orders b then a then c", func(t *testing.T) {
		configContent := "b: b\na: a\nc: c"
		config := make(rawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, rawRepoFundingConfigEntry{"b", "b"}, config[0])
		assert.Equal(t, rawRepoFundingConfigEntry{"a", "a"}, config[1])
		assert.Equal(t, rawRepoFundingConfigEntry{"c", "c"}, config[2])
	})

	t.Run("orders c then a then b", func(t *testing.T) {
		configContent := "c: c\na: a\nb: b"
		config := make(rawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, rawRepoFundingConfigEntry{"c", "c"}, config[0])
		assert.Equal(t, rawRepoFundingConfigEntry{"a", "a"}, config[1])
		assert.Equal(t, rawRepoFundingConfigEntry{"b", "b"}, config[2])
	})

	t.Run("orders c then b then a", func(t *testing.T) {
		configContent := "c: c\nb: b\na: a"
		config := make(rawRepoFundingConfig, 0)
		err := yaml.Unmarshal([]byte(configContent), &config)
		require.NoError(t, err)
		assert.Len(t, config, 3)
		assert.Equal(t, rawRepoFundingConfigEntry{"c", "c"}, config[0])
		assert.Equal(t, rawRepoFundingConfigEntry{"b", "b"}, config[1])
		assert.Equal(t, rawRepoFundingConfigEntry{"a", "a"}, config[2])
	})
}

func TestIsFundingConfig(t *testing.T) {
	cases := []string{
		".forgejo/FUNDING",
		".forgejo/Funding",
		".forgejo/funding",
		".forgejo/fundING",

		".github/FUNDING",
		".github/Funding",
		".github/funding",
		".github/fundING",

		"FUNDING",
		"Funding",
		"funding",
		"fundING",
	}
	for _, fileName := range cases {
		for _, extension := range []string{".yaml", ".yml", ".YML"} {
			path := fileName + extension
			t.Run(fileName, func(t *testing.T) {
				assert.True(t, IsFundingConfig(path))
			})
		}
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
			assert.False(t, IsFundingConfig(fileName))
		})
	}
}

func getFundingFromConfig(t *testing.T, config string) ([]*api.RepoFundingEntry, []error) {
	funding, errs, err := getFundingFromBlob([]byte(config))
	require.NoError(t, err)
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

func assertKoFi(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "ko_fi", expectedText, expectedURL)
}

func assertLiberapay(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "liberapay", expectedText, expectedURL)
}

func assertPatreon(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "patreon", expectedText, expectedURL)
}

func assertTidelift(t *testing.T, entry *api.RepoFundingEntry, expectedText, expectedURL string) {
	t.Helper()
	assertEntry(t, entry, "tidelift", expectedText, expectedURL)
}

func TestFundingConfigParseErrors(t *testing.T) {
	configs := []string{
		`this isn't yaml`,
		`[`,
		`]`,
		`foo`,
		`custom: Arbitrary: text`,
		"custom:\n\t- foo",
	}

	for _, config := range configs {
		data, lineErrors, err := getFundingFromBlob([]byte(config))
		require.Error(t, err)
		require.Nil(t, data)
		require.Nil(t, lineErrors)
		// there's a huge variety of possible YAML parse errors, so we won't bother testing that they're exact.
		// what's important here is that we get an error instead of data.
	}
}

func TestFundingEntriesFromConfig(t *testing.T) {
	defer test.MockProtect(&setting.FundingProviders)()
	setting.LoadBuiltInFundingProviders()

	t.Run("Empty config", func(t *testing.T) {
		funding, errs := getFundingFromConfig(t, "")
		assert.Empty(t, errs)
		assert.Empty(t, funding)
	})

	t.Run("Custom string value", func(t *testing.T) {
		configs := [][3]string{
			{`custom: "https://a.com"`, "https://a.com", "https://a.com"},
			{`custom: https://a.com`, "https://a.com", "https://a.com"},
			{`custom: b.com`, "b.com", "http://b.com"},
			{`custom: "http://withquery.example.com?test=foo"`, "http://withquery.example.com?test=foo", "http://withquery.example.com?test=foo"},
			{`custom: http://withquery.example.com?test=foo`, "http://withquery.example.com?test=foo", "http://withquery.example.com?test=foo"},
			{`custom: "http://thistimewithhash#foo"`, "http://thistimewithhash#foo", "http://thistimewithhash#foo"},
			{`custom: http://thistimewithhash#foo`, "http://thistimewithhash#foo", "http://thistimewithhash#foo"},
			{`custom: https://a.com`, "https://a.com", "https://a.com"},
			{`custom: http://withquery.example.com?test=foo`, "http://withquery.example.com?test=foo", "http://withquery.example.com?test=foo"},
			{`custom: http://thistimewithhash#foo`, "http://thistimewithhash#foo", "http://thistimewithhash#foo"},
			{`custom: 'localhost:1234'`, "localhost:1234", "http://localhost:1234"},
			{`custom: localhost:1234`, "localhost:1234", "http://localhost:1234"},
			{`custom: "https://localhost:8080/"`, "https://localhost:8080/", "https://localhost:8080/"},
			{`custom: "https://[::]:8080/"`, "https://[::]:8080/", "https://[::]:8080/"},
			{`custom: '😀.com'`, "😀.com", "http://xn--e28h.com"},
			// adapted from the list at https://en.wikipedia.org/wiki/International_email:
			{`custom: '例子.广告'`, "例子.广告", "http://xn--fsqu00a.xn--4rr70v"},
			{`custom: 'ಡೇಟಾಮೇಲ್.ಭಾರತ'`, "ಡೇಟಾಮೇಲ್.ಭಾರತ", "http://xn--xscd4bq2d9bd3c.xn--2scrj9c"},
			{`custom: 'डाटा.भारत'`, "डाटा.भारत", "http://xn--c2bd1gb.xn--h2brj9c"},
			{`custom: 'пошта.укр'`, "пошта.укр", "http://xn--80a1acn3a.xn--j1amh"},
			{`custom: 'παράδειγμα.ελ'`, "παράδειγμα.ελ", "http://xn--hxajbheg2az3al.xn--qxam"},
			{`custom: 'Sörensen.example.com'`, "Sörensen.example.com", "http://xn--srensen-90a.example.com"},
			{`custom: 'пример.рф'`, "пример.рф", "http://xn--e1afmkfd.xn--p1ai"},
			{`custom: 'موقع.عر'`, "موقع.عر", "http://xn--4gbrim.xn--wgbp"},
		}
		for _, configCase := range configs {
			config := configCase[0]
			expectedText := configCase[1]
			expectedURL := configCase[2]

			funding, errs := getFundingFromConfig(t, config)
			assert.Empty(t, errs)
			assert.Len(t, funding, 1)
			assertCustom(t, funding[0], expectedText, expectedURL)
		}
	})

	t.Run("Invalid custom string value", func(t *testing.T) {
		configs := []string{
			`custom: 'localhost:text'`,
			`custom: localhost:text`,
			`custom: 'Arbitrary: text'`,
			`custom: "h3://localhost:8080/"`,
			`custom: ' '`,
			`custom: [[test]]`,
			`custom: [["test"]]`,
			`custom: [[]]`,
			`custom:`,
			`custom: 42`,
			"custom: 42\nwhatever:",
		}
		for _, config := range configs {
			funding, errs := getFundingFromConfig(t, config)
			assert.NotEmpty(t, errs)
			assert.Empty(t, funding)
		}
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

func TestFundingEntriesWithErrorsFromConfig(t *testing.T) {
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

	t.Run("Partially invalid (bad key omitted)", func(t *testing.T) {
		config := "liberapay: test\n" +
			"ko_fi:\n" +
			`custom: [test, "https://example.com", 'Arbitrary:4242', 'Arbitrary: 4242', 'h3://localhost']`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 3)
		assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", errs[0].Error())
		assert.Equal(t, `Invalid URL value for key 'custom': parse "http://Arbitrary: 4242": invalid port ": 4242" after host`, errs[1].Error())
		assert.Equal(t, `Invalid URL value for key 'custom': invalid scheme "h3", expected one of: http, https`, errs[2].Error())

		assert.Len(t, funding, 4)
		assertLiberapay(t, funding[0], "liberapay.com/test", "https://liberapay.com/test")
		assertCustom(t, funding[1], "test", "http://test")
		assertCustom(t, funding[2], "https://example.com", "https://example.com")
		assertCustom(t, funding[3], "Arbitrary:4242", "http://arbitrary:4242")
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
			`ko_fi: [42, test, "https://example.com"]`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 3)
		assert.Equal(t, "Unknown funding provider: whatever", errs[0].Error())
		assert.Equal(t, "Invalid type for key 'ko_fi', expected a string or string array", errs[1].Error())
		assert.Equal(t, "Value for key 'ko_fi' does not match pattern /^[^/]+$/", errs[2].Error())

		assert.Len(t, funding, 1)
		assertKoFi(t, funding[0], "ko-fi.com/test", "https://ko-fi.com/test")
	})

	t.Run("Partially invalid (one element of list is bad type)", func(t *testing.T) {
		config := `patreon: [42, "example"]`
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Invalid type for key 'patreon', expected a string or string array", errs[0].Error())

		assert.Len(t, funding, 1)
		assertPatreon(t, funding[0], "patreon.com/example", "https://patreon.com/example")
	})

	t.Run("Partially invalid (too many of one provider)", func(t *testing.T) {
		config := "custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- test5\n" +
			"- test6\n" +
			"- test7\n" +
			"- test8\n" +
			"- test9\n" +
			"- test10\n" +
			"- test11\n" +
			"- test12\n" +
			"- test13\n" +
			"- test14\n" +
			"- test15\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Expected up to 15 funding providers", errs[0].Error())

		assert.Len(t, funding, 15)
		assertCustom(t, funding[0], "test1", "http://test1")
		assertCustom(t, funding[1], "https://example.com", "https://example.com")
		assertCustom(t, funding[2], "test3", "http://test3")
		assertCustom(t, funding[3], "test4", "http://test4")
		assertCustom(t, funding[4], "test5", "http://test5")
		assertCustom(t, funding[5], "test6", "http://test6")
		assertCustom(t, funding[6], "test7", "http://test7")
		assertCustom(t, funding[7], "test8", "http://test8")
		assertCustom(t, funding[8], "test9", "http://test9")
		assertCustom(t, funding[9], "test10", "http://test10")
		assertCustom(t, funding[10], "test11", "http://test11")
		assertCustom(t, funding[11], "test12", "http://test12")
		assertCustom(t, funding[12], "test13", "http://test13")
		assertCustom(t, funding[13], "test14", "http://test14")
		assertCustom(t, funding[14], "test15", "http://test15")
	})

	t.Run("Partially invalid (too many of two providers)", func(t *testing.T) {
		config := "ko_fi: [test]\n" +
			"tidelift: npm/example\n" +
			"custom:\n" +
			"- test1\n" +
			`- "https://example.com"` + "\n" +
			"- test3\n" +
			"- test4\n" +
			"- test5\n" +
			"- test6\n" +
			"- test7\n" +
			"- test8\n" +
			"- test9\n" +
			"- test10\n" +
			"- test11\n" +
			"- test12\n" +
			"- test13\n" +
			"- too_many"
		funding, errs := getFundingFromConfig(t, config)

		assert.Len(t, errs, 1)
		assert.Equal(t, "Expected up to 15 funding providers", errs[0].Error())

		assert.Len(t, funding, 15)
		assertKoFi(t, funding[0], "ko-fi.com/test", "https://ko-fi.com/test")
		assertTidelift(t, funding[1], "tidelift.com/funding/github/npm/example", "https://tidelift.com/funding/github/npm/example")
		assertCustom(t, funding[2], "test1", "http://test1")
		assertCustom(t, funding[3], "https://example.com", "https://example.com")
		assertCustom(t, funding[4], "test3", "http://test3")
		assertCustom(t, funding[5], "test4", "http://test4")
		assertCustom(t, funding[6], "test5", "http://test5")
		assertCustom(t, funding[7], "test6", "http://test6")
		assertCustom(t, funding[8], "test7", "http://test7")
		assertCustom(t, funding[9], "test8", "http://test8")
		assertCustom(t, funding[10], "test9", "http://test9")
		assertCustom(t, funding[11], "test10", "http://test10")
		assertCustom(t, funding[12], "test11", "http://test11")
		assertCustom(t, funding[13], "test12", "http://test12")
		assertCustom(t, funding[14], "test13", "http://test13")
	})
}
