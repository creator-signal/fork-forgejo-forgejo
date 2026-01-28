// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"fmt"

	auth_model "forgejo.org/models/auth"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/authz"
	permissions_errors "forgejo.org/services/permissions/errors"
)

func SetToken(permissionsCtx PermissionsContext, authToken string) error {
	token, err := auth_model.GetAccessTokenBySHA(permissionsCtx.GetContext(), authToken)
	if err != nil {
		return fmt.Errorf("GetAccessTokenBySha: %v", err)
	}

	u, err := user_model.GetUserByID(permissionsCtx.GetContext(), token.UID)
	if err != nil {
		return err
	}

	if err = token.UpdateLastUsed(permissionsCtx.GetContext()); err != nil {
		return fmt.Errorf("UpdateLastUsed: %v", err)
	}

	permissionsCtx.SetDoer(u)
	permissionsCtx.SetToken(token)

	tokenReducer, err := authz.GetAuthorizationReducerForAccessToken(permissionsCtx.GetContext(), token)
	if err != nil {
		return fmt.Errorf("authz.GetAuthorizationReducerForAccessToken: %v", err)
	}

	return SetAuthorization(permissionsCtx, token.Scope, tokenReducer)
}

func SetAuthorization(permissionsCtx PermissionsContext, tokenScope, tokenReducer any) error {
	scope, scopeExists := tokenScope.(auth_model.AccessTokenScope)
	if scopeExists {
		publicOnly, err := scope.PublicOnly()
		if err != nil {
			return permissions_errors.NewServer("parsing public resource scope failed(%v): %v", scope, err)
		}
		permissionsCtx.SetPublicOnly(publicOnly)
	}

	reducer, reducerExists := tokenReducer.(authz.AuthorizationReducer)
	if reducerExists {
		permissionsCtx.SetReducer(reducer)
	} else {
		if permissionsCtx.GetPublicOnly() {
			permissionsCtx.SetReducer(&authz.PublicReposAuthorizationReducer{})
		} else {
			permissionsCtx.SetReducer(&authz.AllAccessAuthorizationReducer{})
		}
	}
	return nil
}

func SetRepository(permissionsCtx PermissionsContext, repoID int64) error {
	if repo := permissionsCtx.GetRepository(); repo != nil && repo.ID == repoID {
		return nil
	}
	repository, err := repo_model.GetRepositoryByID(permissionsCtx.GetContext(), repoID)
	if err != nil {
		return permissions_errors.NewNotFound("repository %d: %v", repoID, err)
	}
	permissionsCtx.SetRepository(repository)
	permission, err := access_model.GetUserRepoPermissionWithReducer(permissionsCtx.GetContext(), permissionsCtx.GetRepository(), permissionsCtx.GetDoer(), permissionsCtx.GetReducer())
	if err != nil {
		return permissions_errors.NewServer("GetUserRepoPermissionWithReducer(%v, %v): %v", permissionsCtx.GetRepository(), permissionsCtx.GetDoer(), err)
	}
	permissionsCtx.SetPermission(&permission)
	return nil
}

// inspired by routers/api/v1/api.go
//
//	func checkTokenPublicOnly() func(ctx *context.APIContext) {
func CheckPublicOnly(permissionsCtx PermissionsContext, categories ...auth_model.AccessTokenScopeCategory) error {
	if !permissionsCtx.GetPublicOnly() || len(categories) == 0 {
		return nil
	}
	switch {
	case auth_model.ContainsCategory(categories, auth_model.AccessTokenScopeCategoryRepository):
		if permissionsCtx.GetRepository() != nil && permissionsCtx.GetRepository().IsPrivate {
			return permissions_errors.NewForbidden("token scope is limited to public repos")
		}
	case auth_model.ContainsCategory(categories, auth_model.AccessTokenScopeCategoryIssue):
		if permissionsCtx.GetRepository() != nil && permissionsCtx.GetRepository().IsPrivate {
			return permissions_errors.NewForbidden("token scope is limited to public issues")
		}
	}
	return nil
}

func RequiresTokenScope(permissionsCtx PermissionsContext, level auth_model.AccessTokenScopeLevel, categories ...auth_model.AccessTokenScopeCategory) error {
	token := permissionsCtx.GetToken()
	requiredScopes := auth_model.GetRequiredScopes(level, categories...)
	allow, err := token.Scope.HasScope(requiredScopes...)
	if err != nil {
		return fmt.Errorf("token.Scope.HasScope(%v): %w", requiredScopes, err)
	}
	if !allow {
		return permissions_errors.NewForbidden("the token does not provide the expected required scopes %v", requiredScopes)
	}
	return CheckPublicOnly(permissionsCtx, categories...)
}
