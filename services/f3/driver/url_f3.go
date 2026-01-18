// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"

	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

func getReviewIDByReviewCommentID(ctx context.Context, tree generic.TreeInterface, pullRequestPath generic.Path, reviewCommentID string) string {
	id := f3_util.ParseInt(reviewCommentID)
	comment, err := issues_model.GetCommentByID(ctx, id)
	if err != nil {
		panic(fmt.Errorf("GetCommentByID(%d): %w", id, err))
	}
	if comment.ReviewID == 0 {
		panic(fmt.Errorf("the comment %d %s has an unexpected reviewID of 0", id, comment.Content))
	}
	return fmt.Sprintf("%d", comment.ReviewID)
}

func buildProjectPath(ctx context.Context, tree generic.TreeInterface, owner, project string) generic.Path {
	ownerPath := buildOwnerPath(ctx, tree, owner)
	if ownerPath == nil {
		return nil
	}
	return ownerPath.SetProjects().SetProject(project)
}

func buildOwnerPath(ctx context.Context, tree generic.TreeInterface, owner string) generic.Path {
	user, err := user_model.GetUserByName(ctx, owner)
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			tree.Error("no organization or user %v", owner)
			return nil
		}
		panic(fmt.Errorf("GetUserByName(%s): %w", owner, err))
	}
	path := generic.NewPathFromString("/").SetForge()
	if user.IsOrganization() {
		return path.SetOrganizations().SetOrganization(user.LowerName)
	}
	return path.SetUsers().SetUser(user.LowerName)
}
