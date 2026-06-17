// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

// Audit holds the configuration of the security audit log.
var Audit = struct {
	Enabled bool
}{
	Enabled: false,
}

func loadAuditFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("audit")
	Audit.Enabled = sec.Key("ENABLED").MustBool(false)
}
