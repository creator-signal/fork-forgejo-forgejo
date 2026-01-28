// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package helpers

import (
	"context"
	"fmt"
	"strings"

	auth_model "forgejo.org/models/auth"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
)

func ToScopeString(scopeLevel auth_model.AccessTokenScopeLevel) string {
	var scopes []string
	for _, scope := range auth_model.GetRequiredScopes(scopeLevel, auth_model.AllAccessTokenScopeCategories...) {
		scopes = append(scopes, string(scope))
	}
	return strings.Join(scopes, ",")
}

func CreateToken(ctx context.Context, user *user_model.User, scopeString string, repoIDs ...int64) (*auth_model.AccessToken, error) {
	scope, err := auth_model.AccessTokenScope(scopeString).Normalize()
	if err != nil {
		return nil, err
	}
	resourceAllRepos := len(repoIDs) == 0
	accessToken := &auth_model.AccessToken{
		UID:              user.ID,
		Name:             util.CryptoRandomString(10),
		Scope:            scope,
		ResourceAllRepos: resourceAllRepos,
	}
	if err := auth_model.NewAccessToken(ctx, accessToken); err != nil {
		return nil, err
	}
	if len(repoIDs) > 0 {
		var resourceRepos []*auth_model.AccessTokenResourceRepo
		for _, repoID := range repoIDs {
			resourceRepos = append(resourceRepos, &auth_model.AccessTokenResourceRepo{
				TokenID: accessToken.ID,
				RepoID:  repoID,
			})
		}
		if err := auth_model.InsertAccessTokenResourceRepos(ctx, accessToken.ID, resourceRepos); err != nil {
			return nil, err
		}
	}
	return accessToken, nil
}

func CreateAdminReadToken(ctx context.Context) (string, func(), error) {
	admins, err := user_model.GetAllAdmins(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("user_model.GetAllAdmins: %w", err)
	}
	if len(admins) == 0 {
		return "", nil, fmt.Errorf("there are no admin users")
	}

	user := admins[0]
	readScope := ToScopeString(auth_model.Read)
	token, err := CreateToken(ctx, user, readScope)
	if err != nil {
		return "", nil, err
	}
	return token.Token, func() {
		if err := auth_model.DeleteAccessTokenByID(ctx, token.ID, user.ID); err != nil {
			log.Error("ignored auth_model.DeleteAccessTokenByID(%v): %w", token.ID, err)
		}
	}, nil
}
