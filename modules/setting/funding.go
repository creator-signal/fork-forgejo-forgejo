// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package setting

import (
	"regexp"
	"strings"

	"forgejo.org/modules/log"
)

// A funding provider, as it appears in the server config
type FundingProviderConfig struct {
	// The name of the funding platform
	Name string

	// A format string that defines a URL, ideally to a given user profile, to
	// which users should be sent to support a project. This string should
	// contain at least one instance of %s or %[1]s, which will be replaced with
	// the string given in a repo's funding config.
	//
	// This is the only required config key; the other details may be derived
	// from this and the platform name.
	URL string

	// A regular expression which input values must match before they may be
	// interpolated into the URL template.
	//
	// The default value permits a single path segment.
	InputPattern *regexp.Regexp

	// A format string that defines the text that should show in place of a URL
	// in the UI. This string should contain at least one instance of %s or
	// %[1]s, which will be replaced with the string given in a repo's funding
	// config.
	//
	// When parsed from the server config, this value defaults to the value of
	// `URL` without the scheme.
	Text string
}

var FundingProviders map[string]*FundingProviderConfig

// The maximum number of funding entries that may be present in a given funding
// config, regardless of how many of those entries share a funding provider.
//
// This limit is arbitrary, and may be changed later, though this limit seems
// reasonable for now.
const MaxFundingEntriesPerConfig = 15

// Ensures that any formatting sigils (%s, etc.) are rendered inert, except for
// %[1]s. Also transforms %s into %[1]s, because these format strings only
// ever receive a single argument, which the template may use in multiple
// places.
func cleanUpSigils(s string) string {
	result := strings.ReplaceAll(s, "%", "%%")             // escape away all sigils
	result = strings.ReplaceAll(result, "%%s", "%[1]s")    // allow %s
	result = strings.ReplaceAll(result, "%%[1]s", "%[1]s") // allow %[1]s
	return result
}

func addFundingProvider(providers map[string]*FundingProviderConfig, provider *FundingProviderConfig) {
	providers[provider.Name] = provider
}

const (
	singleSegmentPattern = `^[^/]+$`               // e.g. "example"
	twoSegmentPattern    = `^[^/]+\/[^/]+$`        // e.g. "a/example"
	threeSegmentPattern  = `^[^/]+\/[^/]+\/[^/]+$` // e.g. "a/b/c"
	anythingPattern      = `.+`                    // e.g. "http://a.com"
)

func LoadBuiltInFundingProviders() {
	FundingProviders = make(map[string]*FundingProviderConfig)

	singleSegmentRegex := regexp.MustCompile(singleSegmentPattern)
	twoSegmentRegex := regexp.MustCompile(twoSegmentPattern)
	threeSegmentRegex := regexp.MustCompile(threeSegmentPattern)
	anythingRegex := regexp.MustCompile(anythingPattern)

	// built-in providers are largely based on github's list at <https://docs.github.com/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository#about-funding-files>
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "community_bridge", // aka LFX Mentorship, but the config calls it community_bridge for compat
		Text:         "funding.communitybridge.org/projects/%[1]s",
		URL:          "https://funding.communitybridge.org/projects/%[1]s", // we might consider using the new URL here if their redirect ever breaks
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "github",
		Text:         "github.com/sponsors/%[1]s",
		URL:          "https://github.com/sponsors/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "issuehunt",
		Text:         "issuehunt.io/r/%[1]s",
		URL:          "https://issuehunt.io/r/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "ko_fi",
		Text:         "ko-fi.com/%[1]s",
		URL:          "https://ko-fi.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "liberapay",
		Text:         "liberapay.com/%[1]s",
		URL:          "https://liberapay.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "open_collective",
		Text:         "opencollective.com/%[1]s",
		URL:          "https://opencollective.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "patreon",
		Text:         "patreon.com/%[1]s",
		URL:          "https://patreon.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "tidelift",
		Text:         "tidelift.com/funding/github/%[1]s",
		URL:          "https://tidelift.com/funding/github/%[1]s",
		InputPattern: twoSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "polar",
		Text:         "polar.sh/%[1]s",
		URL:          "https://polar.sh/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "buy_me_a_coffee",
		Text:         "buymeacoffee.com/%[1]s",
		URL:          "https://buymeacoffee.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "thanks_dev",
		Text:         "thanks.dev/%[1]s",
		URL:          "https://thanks.dev/%[1]s",
		InputPattern: threeSegmentRegex, // we expect something like "u/gh/example"
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "custom",
		Text:         "%[1]s",
		URL:          "%[1]s",
		InputPattern: anythingRegex, // matches anything; the final value is treated like a URL in any case
	})
}

func loadCustomFundingProvidersFrom(rootCfg ConfigProvider) {
	LoadBuiltInFundingProviders()

	const keyText = "TEXT"
	const keyURL = "URL"
	const keyInputPattern = "INPUT_PATTERN"

	for _, sec := range rootCfg.Section("funding").ChildSections() {
		name := strings.TrimPrefix(sec.Name(), "funding.")
		if name == "" {
			log.Warn("name is empty, funding %s ignored", sec.Name())
			continue
		}

		rawText := sec.Key(keyText).MustString("")
		rawURL := sec.Key(keyURL).MustString("")
		rawInputPattern := sec.Key(keyInputPattern).MustString(singleSegmentPattern)

		inputPattern, err := regexp.Compile(rawInputPattern)
		if err != nil {
			singleSegmentRegex := regexp.MustCompile(singleSegmentPattern)
			log.Warn("%s.%s %v, using /%s/ instead", sec.Name(), keyInputPattern, err, singleSegmentRegex.String())
			inputPattern = singleSegmentRegex
		}

		url := cleanUpSigils(rawURL)

		// get the url scheme, if any
		scheme, _, found := strings.Cut(url, "://") // e.g. "https://localhost/%s"
		if !found {
			scheme = ""
		}

		// default text to just the url minus scheme
		text := rawText
		if text == "" {
			text = strings.TrimPrefix(url, scheme+"://")
			// the sigils are already tidy here, no need to clean them up again!
		} else {
			text = cleanUpSigils(text)
		}

		provider := new(FundingProviderConfig)
		provider.Name = name
		provider.Text = text
		provider.URL = url
		provider.InputPattern = inputPattern

		if FundingProviders[name] != nil {
			log.Warn("%s funding provider already exists, existing provider %s is unchanged", sec.Name(), name)
		} else {
			FundingProviders[name] = provider
		}
	}
}

func GetFundingProviderByName(name string) *FundingProviderConfig {
	for _, provider := range FundingProviders {
		if provider.Name == name {
			return provider
		}
	}

	return nil
}
