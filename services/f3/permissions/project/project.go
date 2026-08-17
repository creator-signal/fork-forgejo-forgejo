// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package project

import (
	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions "forgejo.org/services/f3/permissions"
	f3_permissions_api "forgejo.org/services/f3/permissions/api"
)

func Get(repo *repo_model.Repository) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Read, auth_model.AccessTokenScopeCategoryRepository)
		f3_permissions_api.RepoAccess(ctx)
		f3_permissions_api.CheckTokenPublicOnly(ctx, nil)
		f3_permissions_api.ReqAnyRepoReader(ctx)
	}
}

func PutUser(repo *repo_model.Repository) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Write, auth_model.AccessTokenScopeCategoryUser)
		f3_permissions_api.ReqToken(ctx)
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Write, auth_model.AccessTokenScopeCategoryRepository)
	}
}

func PutOrganization(owner *user_model.User, repo *repo_model.Repository) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Write, auth_model.AccessTokenScopeCategoryOrganization)
		f3_permissions_api.CheckTokenPublicOnly(ctx, owner)
		f3_permissions_api.ReqToken(ctx)
	}
}

func PutFork(repo *repo_model.Repository, organizationName *string) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Read, auth_model.AccessTokenScopeCategoryRepository)
		f3_permissions_api.RepoAccess(ctx)
		f3_permissions_api.CheckTokenPublicOnly(ctx, nil)
		f3_permissions_api.ReqToken(ctx)
		f3_permissions_api.ReqRepoReader(ctx, unit.TypeCode)
		f3_permissions_api.CheckForkDestination(ctx, organizationName)
	}
}

func Delete(owner *user_model.User, repo *repo_model.Repository) f3_permissions.CheckFunc {
	return func(ctx *f3_context.F3) {
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		f3_permissions_api.ReqToken(ctx)
		f3_permissions_api.TokenRequiresScopes(ctx, auth_model.Write, auth_model.AccessTokenScopeCategoryRepository)
		f3_permissions_api.RepoAccess(ctx)
		f3_permissions_api.TokenRequiresRepoOwnerScope(ctx, owner, auth_model.Write)
		f3_permissions_api.ReqOwner(ctx, nil)
	}
}
