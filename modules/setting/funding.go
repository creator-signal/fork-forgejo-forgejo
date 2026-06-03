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

	// The max number of entries of this platform which may appear in a repo's
	// funding config. A value of 0 effectively disables the funding option.
	Limit uint

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
	threeSegmentPattern  = `^[^/]+\/[^/]+\/[^/]+$` // e.g. "a/b/c"
	anythingPattern      = `.+`                    // e.g. "http://a.com"
)

func loadCustomFundingProvidersFrom(rootCfg ConfigProvider) {
	FundingProviders = make(map[string]*FundingProviderConfig)

	singleSegmentRegex := regexp.MustCompile(singleSegmentPattern)
	threeSegmentRegex := regexp.MustCompile(threeSegmentPattern)
	anythingRegex := regexp.MustCompile(anythingPattern)

	// built-in providers are largely based on github's list at <https://docs.github.com/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/displaying-a-sponsor-button-in-your-repository#about-funding-files>
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "community_bridge", // aka LFX Mentorship, but the config calls it community_bridge for compat
		Limit:        1,
		Text:         "funding.communitybridge.org/projects/%[1]s",
		URL:          "https://funding.communitybridge.org/projects/%[1]s", // we might consider using the new URL here if their redirect ever breaks
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "github",
		Limit:        4, // this limit feels a bit github-centric, but we'll leave it like this for compat
		Text:         "github.com/sponsors/%[1]s",
		URL:          "https://github.com/sponsors/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "issuehunt",
		Limit:        1,
		Text:         "issuehunt.io/r/%[1]s",
		URL:          "https://issuehunt.io/r/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "ko_fi",
		Limit:        1,
		Text:         "ko-fi.com/%[1]s",
		URL:          "https://ko-fi.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "liberapay",
		Limit:        1,
		Text:         "liberapay.com/%[1]s",
		URL:          "https://liberapay.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "open_collective",
		Limit:        1,
		Text:         "opencollective.com/%[1]s",
		URL:          "https://opencollective.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "patreon",
		Limit:        1,
		Text:         "patreon.com/%[1]s",
		URL:          "https://patreon.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "tidelift",
		Limit:        0,                                   // this provider no longer exists, we include it here for testing the behavior of custom providers with limit 0
		Text:         "tidelift.com/funding/github/%[1]s", // not sure how we'd even handle something like this with github baked in :/
		URL:          "https://tidelift.com/funding/github/%[1]s",
		InputPattern: singleSegmentRegex, // normally takes two segments, but there's no point instantiating a whole Regexp for an impossible case
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "polar",
		Limit:        1,
		Text:         "polar.sh/%[1]s",
		URL:          "https://polar.sh/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "buy_me_a_coffee",
		Limit:        1,
		Text:         "buymeacoffee.com/%[1]s",
		URL:          "https://buymeacoffee.com/%[1]s",
		InputPattern: singleSegmentRegex,
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "thanks_dev",
		Limit:        1,
		Text:         "thanks.dev/%[1]s",
		URL:          "https://thanks.dev/%[1]s",
		InputPattern: threeSegmentRegex, // we expect something like "u/gh/example"
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name:         "custom",
		Limit:        4,
		Text:         "%[1]s",
		URL:          "%[1]s",
		InputPattern: anythingRegex, // matches anything; the final value is treated like a URL in any case
	})

	const keyLimit = "LIMIT"
	const keyText = "TEXT"
	const keyURL = "URL"
	const keyInputPattern = "INPUT_PATTERN"
	const lowerLimit = 0 // a value of 0 effectively disables the provider
	const upperLimit = 16

	for _, sec := range rootCfg.Section("funding").ChildSections() {
		name := strings.TrimPrefix(sec.Name(), "funding.")
		if name == "" {
			log.Warn("name is empty, funding %s ignored", sec.Name())
			continue
		}

		rawLimit := sec.Key(keyLimit).MustInt(1)
		rawText := sec.Key(keyText).MustString("")
		rawURL := sec.Key(keyURL).MustString("")
		rawInputPattern := sec.Key(keyInputPattern).MustString(singleSegmentPattern)

		limit := uint(rawLimit)
		if rawLimit < lowerLimit {
			log.Warn("%s.%s should be no lower than %[3]d, clamping to %[3]d", sec.Name(), keyLimit, lowerLimit)
			limit = lowerLimit
		} else if rawLimit > upperLimit {
			log.Warn("%s.%s should be no higher than %[3]d, clamping to %[3]d", sec.Name(), keyLimit, upperLimit)
			limit = upperLimit
		}

		inputPattern, err := regexp.Compile(rawInputPattern)
		if err != nil {
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
		provider.Limit = limit
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
