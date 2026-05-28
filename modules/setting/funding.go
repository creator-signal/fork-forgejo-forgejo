// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package setting

import (
	"strings"

	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
)

// TODO: this should probably be its own type. In the API, we should send an absolute icon URL in all cases, i.e. IconURL
var FundingProviders map[string]*api.FundingProvider

// Ensures that any formatting sigils (%s, etc.) are rendered inert, except for
// %[1]s. Also, %s is transformed into %[1]s, because these format strings only
// ever receive a single argument, which may be used in multiple places.
func cleanUpSigils(s string) (string) {
	result := strings.ReplaceAll(s, "%", "%%") // escape away all sigils
	result = strings.ReplaceAll(result, "%%s", "%[1]s") // allow %s
	result = strings.ReplaceAll(result, "%%[1]s", "%[1]s") // allow %[1]s
	return result
}

// Removes the last element of the slice.
func popLast(s []string) []string {
	if len(s) > 0 {
		s = s[:len(s) - 1]
	}
	return s
}

func loadCustomFundingProvidersFrom(rootCfg ConfigProvider) {
	// FIXME: The Gitea guys say "suggest uing option method like lable templates. see https://github.com/go-gitea/gitea/blob/main/modules/options/base.go", I'm not sure what that means tho? Maybe ask in my PR whether this or some other way is the "best" approach to adding a new category of app.ini options
	FundingProviders = make(map[string]*api.FundingProvider)

	FundingProviders["custom"] = &api.FundingProvider{
		Name: "custom",
		Limit: 4,
		Text: "%[1]s",
		URL:  "%[1]s",
		Icon: "img/svg/octicon-link.svg", // this value is ignored for Name:custom
		// TODO: we should derive the asset path here; if the file does not exist, warn in the console and leave it empty (template defaults to octicon-heart)
	}

	FundingProviders["ko_fi"] = &api.FundingProvider{
		Name: "ko_fi",
		Limit: 1,
		Text: "ko-fi.com/%[1]s",
		URL:  "https://ko-fi.com/%[1]s",
		Icon: "img/funding/ko_fi.svg",
	}

	FundingProviders["liberapay"] = &api.FundingProvider{
		Name: "liberapay",
		Limit: 1,
		Text: "liberapay.com/%[1]s",
		URL:  "https://liberapay.com/%[1]s",
		Icon: "img/funding/liberapay.svg",
	}

	const keyLimit = "LIMIT"
	const keyText = "TEXT"
	const keyUrl = "URL"
	const keyIcon = "ICON"
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

		limit := uint(1)
		if raw_limit < lowerLimit {
			log.Warn("%s.%s should be no lower than %d, clamping to %d", sec.Name(), keyLimit, lowerLimit, lowerLimit)
			limit = lowerLimit
		} else if raw_limit > upperLimit {
			log.Warn("%s.%s should be no higher than %d, clamping to %d", sec.Name(), keyLimit, upperLimit, upperLimit)
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

		provider := new(api.FundingProvider)
		provider.Name = name
		provider.Limit = limit
		provider.Text = text
		provider.URL = url
		provider.Icon = raw_icon

		if FundingProviders[name] != nil {
			log.Warn("%s funding provider already exists, existing provider %s is unchanged", sec.Name(), name)
		} else {
			FundingProviders[name] = provider
		}
	}
}

func GetFundingProviderByName(name string) *api.FundingProvider {
	for _, provider := range FundingProviders {
		if provider.Name == name {
			return provider
		}
	}

	return nil
}
