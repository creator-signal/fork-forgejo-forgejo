// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

//
// Each compliance suite test variations of
//
// - token
// - reducer
// - public scope
//
// It verifies that the action check succeeds with the expected minimum permissions level
//
// Get - works with perm_model.AccessModeRead or higher
// Put - works with perm_model.AccessModeWrite or higher
// etc.
//
// It also verifies that it fails with a permission error if the permission level is lower
//
// Get - fails with perm_model.AccessModeNone
// Put - fails with perm_model.AccessModeRead or lower
// etc.
//
// Verifying the enforcement of permissions that are specific to a given resource does not
// belong in the compliance suite. It must be verified in a test specific to the resource.
// This is the case, for instance, for verifying that comment IDs belong to the resource being
// commented (because the ID is absolute and not relative)
//

package context

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	permissions_context "forgejo.org/services/permissions/context"
	permissions_errors "forgejo.org/services/permissions/errors"
	permissions_helpers "forgejo.org/services/permissions/helpers"
	repo_service "forgejo.org/services/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func ComplianceRepositoryGet(t *testing.T, buildCtx func() permissions_context.PermissionsContext, repo *repo_model.Repository, check permissions_context.CheckFunc) {
	t.Helper()

	createToken := func(t *testing.T, user *user_model.User, scopeString string, repoIDs ...int64) string {
		t.Helper()
		token, err := permissions_helpers.CreateToken(t.Context(), user, scopeString, repoIDs...)
		require.NoError(t, err)
		return token.Token
	}

	require.False(t, repo.IsPrivate)

	t.Run("Get public repository regular doer PublicOnly scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		require.NoError(t, permissions_context.SetToken(ctx, createToken(t, regularUser, "all,public-only")))
		require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
		assert.NoError(t, check(ctx))
	})

	t.Run("Get public repository regular doer AccessModeRead scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		require.NotEqual(t, regularUser.ID, repo.OwnerID)
		readScope := permissions_helpers.ToScopeString(auth_model.Read)
		require.NoError(t, permissions_context.SetToken(ctx, createToken(t, regularUser, readScope)))
		require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
		assert.NoError(t, check(ctx))
	})

	t.Run("Get public repository regular doer AccessModeNone scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		require.NotEqual(t, regularUser.ID, repo.OwnerID)
		noAccessScope := permissions_helpers.ToScopeString(auth_model.NoAccess)
		require.NoError(t, permissions_context.SetToken(ctx, createToken(t, regularUser, noAccessScope)))
		require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
		assert.ErrorIs(t, check(ctx), permissions_errors.Forbidden)
	})

	repo.IsPrivate = true
	require.NoError(t, repo_service.UpdateRepository(t.Context(), repo, true))

	t.Run("Get private repository owner doer AccessModeRead scope", func(t *testing.T) {
		ctx := buildCtx()
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		readScope := permissions_helpers.ToScopeString(auth_model.Read)
		require.NoError(t, permissions_context.SetToken(ctx, createToken(t, owner, readScope)))
		require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
		assert.NoError(t, check(ctx))
	})

	t.Run("Get private repository owner doer AccessModeRead scope that does not apply to this repository", func(t *testing.T) {
		ctx := buildCtx()
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		unrelatedPrivateRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{IsPrivate: true}, builder.Neq{"`owner_id`": owner.ID, "`id`": repo.ID})
		readScope := permissions_helpers.ToScopeString(auth_model.Read)
		require.NoError(t, permissions_context.SetToken(ctx, createToken(t, owner, readScope, unrelatedPrivateRepo.ID)))
		require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
		assert.ErrorIs(t, check(ctx), permissions_errors.Forbidden)
	})
}
