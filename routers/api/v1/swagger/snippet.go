// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package swagger

import (
	api "forgejo.org/modules/structs"
)

// Snippet
// swagger:response Snippet
type swaggerResponseSnippet struct {
	// in:body
	Body api.Snippet `json:"body"`
}

// SnippetList
// swagger:response SnippetList
type swaggerResponseSnippetList struct {
	// in:body
	Body api.SnippetList `json:"body"`
}

// SnippetFiles
// swagger:response SnippetFiles
type swaggerResponseSnippetFiles struct {
	// in:body
	Body []api.SnippetFile `json:"body"`
}
