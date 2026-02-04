// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"context"

	user_model "forgejo.org/models/user"
)

type SnippetList []*Snippet //revive:disable-line:exported

func (snippetList SnippetList) LoadOwner(ctx context.Context) error {
	ownerCache := make(map[int64]*user_model.User)

	for _, snippet := range snippetList {
		snippet.Owner = ownerCache[snippet.OwnerID]
		if snippet.Owner != nil {
			continue
		}

		err := snippet.LoadOwner(ctx)
		if err != nil {
			return err
		}

		ownerCache[snippet.OwnerID] = snippet.Owner
	}

	return nil
}
