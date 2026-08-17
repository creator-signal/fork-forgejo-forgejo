// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	"fmt"

	auth_model "forgejo.org/models/auth"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	"forgejo.org/services/auth"
	"forgejo.org/services/authz"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions_api "forgejo.org/services/f3/permissions/api"
)

var _ auth.AuthenticationResult = &accessTokenAuthenticationResult{}

type accessTokenAuthenticationResult struct {
	*auth.BaseAuthenticationResult
	user    *user_model.User
	scope   auth_model.AccessTokenScope
	reducer authz.AuthorizationReducer
}

func (r *accessTokenAuthenticationResult) User() *user_model.User {
	return r.user
}

func (r *accessTokenAuthenticationResult) Scope() optional.Option[auth_model.AccessTokenScope] {
	return optional.Some(r.scope)
}

func (r *accessTokenAuthenticationResult) Reducer() authz.AuthorizationReducer {
	return r.reducer
}

func SetToken(ctx *f3_context.F3, authToken string) {
	token, err := auth_model.GetAccessTokenBySHA(ctx.Context(), authToken)
	if err != nil {
		panic(fmt.Errorf("GetAccessTokenBySha: %v", err))
	}

	u, err := user_model.GetUserByID(ctx.Context(), token.UID)
	if err != nil {
		panic(err)
	}

	if err = token.UpdateLastUsed(ctx.Context()); err != nil {
		panic(fmt.Errorf("UpdateLastUsed: %v", err))
	}

	ctx.SetDoer(u)
	ctx.SetIsSigned(true)
	ctx.SetToken(token)

	tokenReducer, err := authz.GetAuthorizationReducerForAccessToken(ctx.Context(), token)
	if err != nil {
		panic(fmt.Errorf("authz.GetAuthorizationReducerForAccessToken: %v", err))
	}

	ctx.SetAuthentication(&accessTokenAuthenticationResult{user: u, scope: token.Scope, reducer: tokenReducer})

	f3_permissions_api.Authorization(ctx)
}
