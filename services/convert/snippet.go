// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package convert

import (
	"context"

	snippet_model "forgejo.org/models/snippet"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
)

// ToSnippet convert a snippet_model.Snippet to an api.Snippet
func ToSnippet(ctx context.Context, snippet *snippet_model.Snippet, doer *user_model.User) *api.Snippet {
	result := &api.Snippet{
		ID:          snippet.ID,
		UUID:        snippet.UUID,
		Name:        snippet.Name,
		Description: snippet.Description,
		Visibility:  snippet.Visibility.String(),
		Language:    snippet.Language,
		Created:     snippet.CreatedUnix.AsTime(),
		Updated:     snippet.UpdatedUnix.AsTime(),
	}

	if snippet.Owner != nil {
		result.Owner = ToUser(ctx, snippet.Owner, doer)
	}

	return result
}

// ToSnippetList convert a snippet_model.SnippetList to an api.SnippetList
func ToSnippetList(ctx context.Context, snippedList snippet_model.SnippetList, doer *user_model.User) *api.SnippetList {
	newList := make([]*api.Snippet, len(snippedList))

	for pos, snippet := range snippedList {
		newList[pos] = ToSnippet(ctx, snippet, doer)
	}

	return &api.SnippetList{Snippets: newList}
}
