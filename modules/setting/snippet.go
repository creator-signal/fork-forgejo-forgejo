// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"path"
	"path/filepath"
)

const maxFilesDefault = 3

var Snippet = struct {
	Enabled            bool
	RootPath           string
	MaxFilesPerSnippet int
}{
	Enabled:            true,
	RootPath:           "",
	MaxFilesPerSnippet: maxFilesDefault,
}

func loadSnippetFrom(rootCfg ConfigProvider) {
	sec := rootCfg.Section("snippet")
	Snippet.Enabled = sec.Key("ENABLED").MustBool(true)
	Snippet.RootPath = sec.Key("ROOT").MustString(path.Join(AppDataPath, "snippets"))
	if !filepath.IsAbs(Snippet.RootPath) {
		Snippet.RootPath = filepath.Join(AppWorkPath, Snippet.RootPath)
	} else {
		Snippet.RootPath = filepath.Clean(Snippet.RootPath)
	}
	Snippet.MaxFilesPerSnippet = sec.Key("MAX_FILES_PER_SNIPPET").MustInt(maxFilesDefault)
}
