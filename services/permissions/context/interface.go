// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"context"

	auth_model "forgejo.org/models/auth"
	org_model "forgejo.org/models/organization"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/auth"
	"forgejo.org/services/authz"
)

type PermissionsContext interface {
	GetContext() context.Context

	GetToken() *auth_model.AccessToken
	SetToken(*auth_model.AccessToken)

	GetRepository() *repo_model.Repository
	SetRepository(*repo_model.Repository)

	GetPermission() *access_model.Permission
	SetPermission(*access_model.Permission)

	GetDoer() *user_model.User
	SetDoer(*user_model.User)

	GetIsSigned() bool
	SetIsSigned(bool)

	GetUser() *user_model.User
	SetUser(*user_model.User)

	GetOrg() *org_model.Organization
	SetOrg(*org_model.Organization)

	GetTeam() *org_model.Team
	SetTeam(*org_model.Team)

	GetPackageOwner() *user_model.User
	SetPackageOwner(*user_model.User)

	GetPublicOnly() bool
	SetPublicOnly(bool)

	GetReducer() authz.AuthorizationReducer
	SetReducer(authz.AuthorizationReducer)

	GetAuthentication() auth.AuthenticationResult
	SetAuthentication(auth.AuthenticationResult)

	GetRequiredScopeCategories() []auth_model.AccessTokenScopeCategory
	SetRequiredScopeCategories([]auth_model.AccessTokenScopeCategory)

	Error(status int, title string, obj any)
	InternalServerError(err error)
	NotFound(objs ...any)
	WrittenStatus() int
}
