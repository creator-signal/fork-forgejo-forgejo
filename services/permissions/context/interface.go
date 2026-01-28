// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"context"

	auth_model "forgejo.org/models/auth"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/authz"
)

type CheckFunc func(PermissionsContext) error

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

	GetPublicOnly() bool
	SetPublicOnly(bool)

	GetReducer() authz.AuthorizationReducer
	SetReducer(authz.AuthorizationReducer)
}
