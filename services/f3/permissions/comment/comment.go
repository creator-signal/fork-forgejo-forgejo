// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package comment

import (
	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions "forgejo.org/services/f3/permissions"
	f3_permissions_api "forgejo.org/services/f3/permissions/api"
	f3_permissions_errors "forgejo.org/services/f3/permissions/errors"
)

func Get(repoID, issueID int64, comment *issues_model.Comment) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions.SetRepositoryFromID(ctx, repoID)
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Read, auth_model.AccessTokenScopeCategoryIssue)
		f3_permissions_api.RepoAccess(ctx)
		f3_permissions_api.CheckTokenPublicOnly(ctx, nil)
		f3_permissions_api.MustEnableIssuesOrPulls(ctx)
		if err := comment.LoadIssue(ctx.Context()); err != nil {
			panic(f3_permissions_errors.NewServer("comment.LoadIssue(%d): %w", comment.ID, err))
		}
		f3_permissions_api.ReqValidCommentID(ctx, issueID, comment)
		comment.Issue.Repo = ctx.Repository()
	}
}
