// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"strings"

	"forgejo.org/modules/highlight"
	api "forgejo.org/modules/structs"
)

type SnippetFiles []*api.SnippetFile //revive:disable-line:exported

func (files SnippetFiles) Contains(name string) bool {
	for _, currentFile := range files {
		if strings.EqualFold(name, currentFile.Name) {
			return true
		}
	}

	return false
}

func (files SnippetFiles) Highlight() error {
	var err error

	for _, currentFile := range files {
		currentFile.HighlightedContent, _, err = highlight.File(currentFile.Name, "", []byte(currentFile.Content))
		if err != nil {
			return err
		}
	}

	return nil
}
