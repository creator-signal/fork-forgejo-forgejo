// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"fmt"
	"strings"

	"forgejo.org/modules/analyze"
	"forgejo.org/modules/highlight"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/util"
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

func (files SnippetFiles) ValidateNames() error {
	for _, currentFile := range files {
		if util.PathContainsDirectory(currentFile.Name) {
			return fmt.Errorf("invalid filename: %s", currentFile.Name)
		}
	}

	return nil
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

func (files SnippetFiles) GetLanguage() string {
	language := ""

	for _, currentFile := range files {
		currentLanguage := analyze.GetCodeLanguage(currentFile.Name, []byte(currentFile.Content))

		// If we have multiple files with a different language, we can't detect a single language
		if language != "" && language != currentLanguage {
			return ""
		}

		language = currentLanguage
	}

	return language
}
