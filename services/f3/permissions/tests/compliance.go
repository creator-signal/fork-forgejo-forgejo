// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions "forgejo.org/services/f3/permissions"
	f3_permissions_helpers "forgejo.org/services/f3/permissions/helpers"
	repo_service "forgejo.org/services/repository"

	f3_assert "code.forgejo.org/f3/gof3/v3/util/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func CompliancePublicRepositoryGet(t *testing.T, buildCtx func() *f3_context.F3, repo *repo_model.Repository, check f3_permissions.CheckFunc) {
	t.Helper()

	createToken := func(t *testing.T, user *user_model.User, scopeString string, repoIDs ...int64) string {
		t.Helper()
		token, err := f3_permissions_helpers.CreateToken(t.Context(), user, scopeString, repoIDs...)
		require.NoError(t, err)
		return token.Token
	}

	require.False(t, repo.IsPrivate)

	t.Run("Get public repository regular doer PublicOnly scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		f3_permissions.SetToken(ctx, createToken(t, regularUser, "all,public-only"))
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		check(ctx)
	})

	t.Run("Get public repository regular doer AccessModeRead scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		require.NotEqual(t, regularUser.ID, repo.OwnerID)
		readScope := f3_permissions_helpers.ToScopeString(auth_model.Read)
		f3_permissions.SetToken(ctx, createToken(t, regularUser, readScope))
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		check(ctx)
	})

	t.Run("Get public repository regular doer AccessModeNone scope", func(t *testing.T) {
		ctx := buildCtx()
		regularUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		require.NotEqual(t, regularUser.ID, repo.OwnerID)
		noAccessScope := f3_permissions_helpers.ToScopeString(auth_model.NoAccess)
		f3_permissions.SetToken(ctx, createToken(t, regularUser, noAccessScope))
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		f3_assert.PanicErrorContains(t, func() { check(ctx) }, "Forbidden: tokenRequiresScope: token does not have at least one of required scope(s): [read:issue]")
	})

	repo.IsPrivate = true
	require.NoError(t, repo_service.UpdateRepository(t.Context(), repo, true))

	t.Run("Get private repository owner doer AccessModeRead scope", func(t *testing.T) {
		ctx := buildCtx()
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		readScope := f3_permissions_helpers.ToScopeString(auth_model.Read)
		f3_permissions.SetToken(ctx, createToken(t, owner, readScope))
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		check(ctx)
	})

	t.Run("Get private repository owner doer AccessModeRead scope that does not apply to this repository", func(t *testing.T) {
		ctx := buildCtx()
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})
		unrelatedPrivateRepo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{IsPrivate: true}, builder.Neq{"`owner_id`": owner.ID, "`id`": repo.ID})
		readScope := f3_permissions_helpers.ToScopeString(auth_model.Read)
		f3_permissions.SetToken(ctx, createToken(t, owner, readScope, unrelatedPrivateRepo.ID))
		f3_permissions.SetRepositoryFromID(ctx, repo.ID)
		f3_assert.PanicErrorContains(t, func() { check(ctx) }, "NotFound")
	})
}
