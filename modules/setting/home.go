// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

// Home settings
var Home = struct {
	ConfigFile string
}{}

func loadHomeFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("home")
	Home.ConfigFile = sec.Key("CONFIG_FILE").MustString("")
}
