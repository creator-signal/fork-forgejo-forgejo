// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"context"
	"fmt"

	"forgejo.org/modules/git"
)

func IsRefProtected(ctx context.Context, repoID int64, ref git.RefName) (bool, error) {
	if ref.IsBranch() {
		result, err := IsBranchProtected(ctx, repoID, ref.BranchName())
		if err != nil {
			return false, fmt.Errorf("could not determine whether branch %q of repository %d is protected: %w",
				ref.BranchName(), repoID, err)
		}
		return result, nil
	}
	if ref.IsTag() {
		protectedTags, err := GetProtectedTags(ctx, repoID)
		if err != nil {
			return false, fmt.Errorf("could not get protected tags of repository %d: %w", repoID, err)
		}
		for _, protectedTag := range protectedTags {
			result, err := protectedTag.Affects(ref)
			if err != nil {
				return false, fmt.Errorf("could not determine whether tag %q of repository %d is protected: %w",
					ref.TagName(), repoID, err)
			}
			if result {
				return result, nil
			}
		}
	}

	return false, nil
}
