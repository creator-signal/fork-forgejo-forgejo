// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package setting

import (
	"fmt"
	"strings"

	"forgejo.org/modules/log"
)

// A funding provider, as it appears in the server config
type FundingProviderConfig struct {
	// The name of the funding platform
	Name     string

	// The max number of entries of this platform which may appear in a repo's
	// funding config. A value of 0 effectively disables the funding option.
	Limit    uint

	// A format string that defines a URL, ideally to a given user profile, to
	// which users should be sent to support a project. This string should
	// contain at least one instance of %s or %[1]s, which will be replaced with
	// the string given in a repo's funding config.
	//
	// This is the only required config key; the other details may be derived
	// from this and the platform name.
	URL      string

	// A format string that defines the text that should show in place of a URL
	// in the UI. This string should contain at least one instance of %s or
	// %[1]s, which will be replaced with the string given in a repo's funding
	// config.
	//
	// When parsed from the server config, this value defaults to the value of
	// `URL` without the scheme.
	Text     string

	// The name of an icon file that identifies the platform. When parsed from
	// the server config, this value defaults to `{Name}.svg`. For custom funding
	// providers, add a small square image or vector to
	// public/assets/img/funding/provider_name.svg. If the file extension differs
	// from svg, or the provider_name differs from the platform name, set the
	// actual filename here.
	IconName string

	// Like `IconName` but only shown in dark theme. `IconName` is used if this
	// value is empty. Note that a default fallback icon is used if `IconName` is
	// not given, NOT `IconNameDark`.
	IconNameDark string
}

var FundingProviders map[string]*FundingProviderConfig

// Ensures that any formatting sigils (%s, etc.) are rendered inert, except for
// %[1]s. Also transforms %s into %[1]s, because these format strings only
// ever receive a single argument, which may be used in multiple places.
func cleanUpSigils(s string) (string) {
	result := strings.ReplaceAll(s, "%", "%%") // escape away all sigils
	result = strings.ReplaceAll(result, "%%s", "%[1]s") // allow %s
	result = strings.ReplaceAll(result, "%%[1]s", "%[1]s") // allow %[1]s
	return result
}

// Returns the path to the provider's icon
func IconForProvider(p *FundingProviderConfig) (string) {
	if p.Name == "custom" {
		return fmt.Sprintf("%s/assets/img/svg/octicon-link.svg", AppSubURL)
	} else if p.IconName == "" {
		return fmt.Sprintf("%s/assets/img/funding/%s.svg", AppSubURL, p.Name)
	} else {
		return fmt.Sprintf("%s/assets/img/funding/%s", AppSubURL, p.IconName)
	}
}

// Returns the path to the provider's dark-theme icon, if any
func DarkIconForProvider(p *FundingProviderConfig) (string) {
	if p.IconNameDark == "" {
		return ""
	} else {
		return fmt.Sprintf("%s/assets/img/funding/%s", AppSubURL, p.IconNameDark)
	}
}

func addFundingProvider(providers map[string]*FundingProviderConfig, provider *FundingProviderConfig) {
	providers[provider.Name] = provider
}

func loadCustomFundingProvidersFrom(rootCfg ConfigProvider) {
	FundingProviders = make(map[string]*FundingProviderConfig)

	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "community_bridge",
		Limit: 1,
		Text: "funding.communitybridge.org/projects/%[1]s",
		URL:  "https://funding.communitybridge.org/projects/%[1]s",
		IconName: "community_bridge.svg",
		IconNameDark: "community_bridge_dark.svg",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "github",
		Limit: 4, // this much feels a bit github-centric, but I'll leave it for compat
		Text: "github.com/sponsors/%[1]s",
		URL:  "https://github.com/sponsors/%[1]s",
		IconName: "github.svg",
		IconNameDark: "github_dark.svg",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "issuehunt",
		Limit: 1,
		Text: "issuehunt.io/r/%[1]s",
		URL:  "https://issuehunt.io/r/%[1]s",
		IconName: "issuehunt.svg",
		IconNameDark: "",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "ko_fi",
		Limit: 1,
		Text: "ko-fi.com/%[1]s",
		URL:  "https://ko-fi.com/%[1]s",
		IconName: "ko_fi.svg",
		IconNameDark: "",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "liberapay",
		Limit: 1,
		Text: "liberapay.com/%[1]s",
		URL:  "https://liberapay.com/%[1]s",
		IconName: "liberapay.svg",
		IconNameDark: "",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "open_collective",
		Limit: 1,
		Text: "opencollective.com/%[1]s",
		URL:  "https://opencollective.com/%[1]s",
		IconName: "open_collective.svg",
		IconNameDark: "",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "patreon",
		Limit: 1,
		Text: "patreon.com/%[1]s",
		URL:  "https://patreon.com/%[1]s",
		IconName: "patreon.svg",
		IconNameDark: "patreon_dark.svg",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "polar",
		Limit: 1,
		Text: "polar.sh/%[1]s",
		URL:  "https://polar.sh/%[1]s",
		IconName: "polar.png",
		IconNameDark: "polar_dark.png",
	})
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "buy_me_a_coffee",
		Limit: 1,
		Text: "buymeacoffee.com/%[1]s",
		URL:  "https://buymeacoffee.com/%[1]s",
		IconName: "buy_me_a_coffee.svg",
		IconNameDark: "buy_me_a_coffee_dark.svg",
	})
	// TODO: thanks.dev
	addFundingProvider(FundingProviders, &FundingProviderConfig{
		Name: "custom",
		Limit: 4,
		Text: "%[1]s",
		URL:  "%[1]s",
		IconName: "", // the template ignores the configured icon for "custom" specifically
		IconNameDark: "",
	})

	const keyLimit = "LIMIT"
	const keyText = "TEXT"
	const keyUrl = "URL"
	const keyIcon = "ICON"
	const keyIconDark = "ICON_DARK"
	const lowerLimit = 0 // a value of 0 effectively disables the provider
	const upperLimit = 16

	for _, sec := range rootCfg.Section("funding").ChildSections() {
		name := strings.TrimPrefix(sec.Name(), "funding.")
		if name == "" {
			log.Warn("name is empty, funding %s ignored", sec.Name())
			continue
		}

		raw_limit := sec.Key(keyLimit).MustInt(1)
		raw_text := sec.Key(keyText).MustString("")
		raw_url := sec.Key(keyUrl).MustString("")
		raw_icon := sec.Key(keyIcon).MustString("")
		raw_icon_dark := sec.Key(keyIconDark).MustString("")

		limit := uint(1)
		if raw_limit < lowerLimit {
			log.Warn("%s.%s should be no lower than %[3]d, clamping to %[3]d", sec.Name(), keyLimit, lowerLimit)
			limit = lowerLimit
		} else if raw_limit > upperLimit {
			log.Warn("%s.%s should be no higher than %[3]d, clamping to %[3]d", sec.Name(), keyLimit, upperLimit)
			limit = upperLimit
		} else {
			limit = uint(raw_limit)
		}

		url := cleanUpSigils(raw_url)

		// get the url scheme, if any
		scheme, _, found := strings.Cut(url, "://") // e.g. "https://localhost/%s"
		if !found {
			scheme = ""
		}

		// default text to just the url minus scheme
		text := raw_text
		if text == "" {
			text = strings.TrimPrefix(url, scheme + "://")
			// the sigils are already tidy here, no need to clean them up again!
		} else {
			text = cleanUpSigils(text)
		}

		// make the icon safe to use as a path segment
		icon := raw_icon
		if icon == "" {
			icon = fmt.Sprintf("%s.svg", name)
		} else if strings.Contains(icon, "..") || strings.ContainsAny(icon, "/\\") {
			icon = fmt.Sprintf("%s.svg", name)
			log.Warn("%s.%s must be a valid filename in public/assets/img/funding, using %s instead", sec.Name(), keyIcon, icon)
		}

		icon_dark := raw_icon_dark
		if icon_dark != "" && strings.Contains(icon_dark, "..") || strings.ContainsAny(icon_dark, "/\\") {
			icon_dark = ""
			log.Warn("%s.%s must be a valid filename in public/assets/img/funding, value is ignored", sec.Name(), keyIconDark)
		}

		provider := new(FundingProviderConfig)
		provider.Name = name
		provider.Limit = limit
		provider.Text = text
		provider.URL = url
		provider.IconName = icon
		provider.IconNameDark = icon_dark

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
