// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	repo_model "forgejo.org/models/repo"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions_errors "forgejo.org/services/f3/permissions/errors"
)

func SetRepositoryFromID(ctx *f3_context.F3, repoID int64) {
	if repo := ctx.Repository(); repo != nil && repo.ID == repoID {
		return
	}
	repository, err := repo_model.GetRepositoryByID(ctx.Context(), repoID)
	if err != nil {
		panic(f3_permissions_errors.NewNotFound("repository %d: %v", repoID, err))
	}
	ctx.SetRepository(repository)
}
