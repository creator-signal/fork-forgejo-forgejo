// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package comment

import (
	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	permissions_context "forgejo.org/services/permissions/context"
	permissions_errors "forgejo.org/services/permissions/errors"
)

func Get(repoID, issueID int64, comment *issues_model.Comment) permissions_context.CheckFunc {
	return func(permissionsCtx permissions_context.PermissionsContext) error {
		if err := comment.LoadIssue(permissionsCtx.GetContext()); err != nil {
			return permissions_errors.NewServer("comment.LoadIssue(%d): %w", comment.ID, err)
		}
		if comment.IssueID != issueID || comment.Issue.RepoID != repoID {
			return permissions_errors.NewNotFound("comment %d repoID %d=%d issueID %d=%d", comment.ID, comment.Issue.RepoID, repoID, comment.IssueID, issueID)
		}
		if err := permissions_context.SetRepository(permissionsCtx, repoID); err != nil {
			return err
		}
		if err := permissions_context.RequiresTokenScope(permissionsCtx, auth_model.Read, auth_model.AccessTokenScopeCategoryRepository, auth_model.AccessTokenScopeCategoryIssue); err != nil {
			return err
		}
		if !permissionsCtx.GetPermission().CanReadIssuesOrPulls(comment.Issue.IsPull) {
			return permissions_errors.NewForbidden("comment %d", comment.ID)
		}
		return nil
	}
}
