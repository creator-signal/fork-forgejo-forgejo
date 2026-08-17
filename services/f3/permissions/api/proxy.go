// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package api

import (
	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
	f3_permissions_errors "forgejo.org/services/f3/permissions/errors"
)

func Authorization(ctx apiv1_permissions.Context) {
	proxy(ctx, apiv1_permissions.APIAuthorization)
}

func RepoAccess(ctx apiv1_permissions.Context) {
	proxy(ctx, apiv1_permissions.RepoAccess)
}

func CheckTokenPublicOnly(ctx apiv1_permissions.Context, owner *user_model.User) {
	var user, org *user_model.User
	if owner != nil && owner.IsOrganization() {
		org = owner
	} else {
		user = owner
	}
	apiv1_permissions.CheckTokenPublicOnly(ctx, user, org, nil)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.CheckTokenPublicOnly)
	}
}

func ReqRepoReader(ctx apiv1_permissions.Context, unitType unit.Type) {
	apiv1_permissions.ReqRepoReader(ctx, unitType)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.ReqRepoReader, unitType)
	}
}

func ReqOwner(ctx apiv1_permissions.Context, unitTypes []unit.Type) {
	apiv1_permissions.ReqOwner(ctx, unitTypes)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.ReqOwner, unitTypes)
	}
}

func CheckForkDestination(ctx apiv1_permissions.Context, organizationName *string) {
	apiv1_permissions.CheckForkDestination(ctx, organizationName)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.CheckForkDestination)
	}
}

func ReqAnyRepoReader(ctx apiv1_permissions.Context) {
	proxy(ctx, apiv1_permissions.ReqAnyRepoReader)
}

func MustEnableIssuesOrPulls(ctx apiv1_permissions.Context) {
	proxy(ctx, apiv1_permissions.MustEnableIssuesOrPulls)
}

func ReqToken(ctx apiv1_permissions.Context) {
	proxy(ctx, apiv1_permissions.ReqToken)
}

func TokenRequiresScopes(ctx apiv1_permissions.Context, level auth_model.AccessTokenScopeLevel, categories ...auth_model.AccessTokenScopeCategory) {
	apiv1_permissions.TokenRequiresScopes(ctx, categories, level)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.TokenRequiresScopes, categories)
	}
}

func TokenRequiresRepoOwnerScope(ctx apiv1_permissions.Context, owner *user_model.User, level auth_model.AccessTokenScopeLevel) {
	apiv1_permissions.TokenRequiresRepoOwnerScope(ctx, owner, level)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.TokenRequiresRepoOwnerScope)
	}
}

func ReqValidCommentID(ctx apiv1_permissions.Context, issueID int64, comment *issues_model.Comment) {
	if comment.IssueID != issueID {
		panic(f3_permissions_errors.NewNotFound("comment %d issueID %d=%d", comment.ID, comment.Issue.RepoID, comment.IssueID, issueID))
	}
	apiv1_permissions.ReqValidCommentID(ctx, comment)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.ReqValidCommentID)
	}
}

func MustEnableLocalIssuesIfIsIssue(ctx apiv1_permissions.Context, issue *issues_model.Issue) {
	apiv1_permissions.MustEnableLocalIssuesIfIsIssue(ctx, issue)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
	if setting.IsInTesting {
		AddPermissionsCheckCall(apiv1_permissions.MustEnableLocalIssuesIfIsIssue)
	}
}

func proxy(ctx apiv1_permissions.Context, f func(apiv1_permissions.Context)) {
	panicIfStatusSet(ctx, f)
	if setting.IsInTesting {
		AddPermissionsCheckCall(f)
	}
}

func panicIfStatusSet(ctx apiv1_permissions.Context, f func(apiv1_permissions.Context)) {
	f(ctx)
	if ctx.WrittenStatus() != 0 {
		panic(ctx.GetError())
	}
}
