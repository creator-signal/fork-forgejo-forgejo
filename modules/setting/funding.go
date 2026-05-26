// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPLv3-or-later

package setting

import (
	"strings"

	"forgejo.org/modules/log"
	api "forgejo.org/modules/structs"
)

var FundingProviders []*api.FundingProvider

func loadBuiltinFundingProviders() {
	FundingProviders = append(FundingProviders, &api.FundingProvider{
		Name: "custom",
		Limit: 4,
		Text: "%s",
		URL:  "%s",
		Icon: "img/svg/octicon-link.svg", // this value is ignored for Name:custom
	})

	FundingProviders = append(FundingProviders, &api.FundingProvider{
		Name: "ko_fi",
		Limit: 1,
		Text: "ko-fi.com/%s",
		URL:  "https://ko-fi.com/%s",
		Icon: "img/funding/ko_fi.svg",
	})
}

func loadCustomFundingProvidersFrom(rootCfg ConfigProvider) {
	for _, sec := range rootCfg.Section("funding").ChildSections() {
		name := strings.TrimPrefix(sec.Name(), "funding.")
		if name == "" {
			log.Warn("name is empty, funding " + sec.Name() + "ignored")
			continue
		}

		provider := new(api.FundingProvider)
		provider.Name = name
		provider.Limit = sec.Key("Limit").MustUint(1)
		provider.Text = sec.Key("Text").MustString("")
		provider.URL = sec.Key("URL").MustString("")
		provider.Icon = sec.Key("Icon").MustString("")

		FundingProviders = append(FundingProviders, provider)
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
