// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"strings"
	"testing"

	"forgejo.org/models/auth"
	"forgejo.org/models/unittest"
	"forgejo.org/services/authz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAuthorizedIntegration(t *testing.T) {
	t.Run("valid - all access", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: true,
			Scope:            auth.AccessTokenScopeReadRepository,
			UI:               auth.AuthorizedIntegrationUIGeneric,
		}
		err := ValidateAuthorizedIntegration(ai, nil)
		require.NoError(t, err)
	})

	t.Run("valid - specified repos", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: false,
			Scope:            auth.AccessTokenScopeReadRepository,
			UI:               auth.AuthorizedIntegrationUIGeneric,
		}
		resources := []*auth.AuthorizedIntegResourceRepo{{RepoID: 12}}
		err := ValidateAuthorizedIntegration(ai, resources)
		require.NoError(t, err)
	})

	t.Run("invalid - no specified repos", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: false,
			Scope:            auth.AccessTokenScopeReadRepository,
			UI:               auth.AuthorizedIntegrationUIGeneric,
		}
		resources := []*auth.AuthorizedIntegResourceRepo{}
		err := ValidateAuthorizedIntegration(ai, resources)
		require.ErrorIs(t, err, authz.ErrSpecifiedReposNone)
	})

	t.Run("invalid - specified repos & public-only", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: false,
			Scope:            auth.AccessTokenScope(strings.Join([]string{string(auth.AccessTokenScopePublicOnly), string(auth.AccessTokenScopeReadRepository)}, ",")),
			UI:               auth.AuthorizedIntegrationUIGeneric,
		}
		resources := []*auth.AuthorizedIntegResourceRepo{{RepoID: 12}}
		err := ValidateAuthorizedIntegration(ai, resources)
		require.ErrorIs(t, err, authz.ErrSpecifiedReposNoPublicOnly)
	})

	t.Run("invalid - specified repos unsupported scopes", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: false,
			Scope:            auth.AccessTokenScopeReadAdmin,
			UI:               auth.AuthorizedIntegrationUIGeneric,
		}
		resources := []*auth.AuthorizedIntegResourceRepo{{RepoID: 12}}
		err := ValidateAuthorizedIntegration(ai, resources)
		require.ErrorIs(t, err, authz.ErrSpecifiedReposInvalidScope)
		require.ErrorContains(t, err, string(auth.AccessTokenScopeReadAdmin))
	})

	t.Run("invalid - missing UI", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			ResourceAllRepos: false,
			Scope:            auth.AccessTokenScopeReadAdmin,
		}
		resources := []*auth.AuthorizedIntegResourceRepo{{RepoID: 12}}
		err := ValidateAuthorizedIntegration(ai, resources)
		require.ErrorIs(t, err, ErrAuthorizedIntegrationBadUI)
		require.ErrorContains(t, err, "invalid UI: \"\"")
	})
}

func TestInsertAuthorizedIntegration(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("success inserts w/ repos", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			UserID:           2,
			UI:               auth.AuthorizedIntegrationUIGeneric,
			ResourceAllRepos: false,
			ClaimRules:       &auth.ClaimRules{},
			Name:             " Magical AI ",
			Scope:            auth.AccessTokenScopeReadRepository,
		}
		rr := []*auth.AuthorizedIntegResourceRepo{
			{
				RepoID: 2,
			},
		}

		err := InsertAuthorizedIntegration(t.Context(), ai, rr)
		require.NoError(t, err)

		fromDB := unittest.AssertExistsAndLoadBean(t, &auth.AuthorizedIntegration{ID: ai.ID})
		assert.Equal(t, "Magical AI", fromDB.Name)

		// IntegID should have been initialized and the repo-specific record saved
		res := unittest.AssertExistsAndLoadBean(t, &auth.AuthorizedIntegResourceRepo{IntegID: ai.ID})
		assert.EqualValues(t, 2, res.RepoID)
	})

	t.Run("validates data", func(t *testing.T) {
		ai := &auth.AuthorizedIntegration{
			UserID:           2,
			UI:               auth.AuthorizedIntegrationUIGeneric,
			ResourceAllRepos: false,
			ClaimRules:       &auth.ClaimRules{},
			Name:             " Magical AI ",
		}
		err := InsertAuthorizedIntegration(t.Context(), ai, nil)
		require.ErrorIs(t, err, authz.ErrSpecifiedReposNone)
	})
}

func TestUpdateAuthorizedIntegration(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	prep := func(t *testing.T) (*auth.AuthorizedIntegration, []*auth.AuthorizedIntegResourceRepo) {
		ai := &auth.AuthorizedIntegration{
			UserID:           2,
			UI:               auth.AuthorizedIntegrationUIGeneric,
			ResourceAllRepos: false,
			ClaimRules:       &auth.ClaimRules{},
			Name:             " Magical AI ",
			Scope:            auth.AccessTokenScopeReadRepository,
		}
		rr := []*auth.AuthorizedIntegResourceRepo{
			{
				RepoID: 2,
			},
		}
		err := InsertAuthorizedIntegration(t.Context(), ai, rr)
		require.NoError(t, err)
		return ai, rr
	}

	t.Run("update basic fields", func(t *testing.T) {
		ai, rr := prep(t)
		ai.Description = "This is the description field."

		err := UpdateAuthorizedIntegration(t.Context(), ai, rr)
		require.NoError(t, err)

		fromDB := unittest.AssertExistsAndLoadBean(t, &auth.AuthorizedIntegration{ID: ai.ID})
		assert.Equal(t, "Magical AI", fromDB.Name)
		assert.Equal(t, "This is the description field.", fromDB.Description)
		unittest.AssertCount(t, &auth.AuthorizedIntegResourceRepo{IntegID: ai.ID}, 1)
	})

	t.Run("update remove resource repos", func(t *testing.T) {
		ai, _ := prep(t)
		ai.ResourceAllRepos = true

		err := UpdateAuthorizedIntegration(t.Context(), ai, nil)
		require.NoError(t, err)

		unittest.AssertCount(t, &auth.AuthorizedIntegResourceRepo{IntegID: ai.ID}, 0)
	})

	t.Run("update add resource repos", func(t *testing.T) {
		ai, _ := prep(t)
		rr := []*auth.AuthorizedIntegResourceRepo{
			{
				RepoID: 2,
			},
			{
				RepoID: 3,
			},
		}

		err := UpdateAuthorizedIntegration(t.Context(), ai, rr)
		require.NoError(t, err)

		unittest.AssertCount(t, &auth.AuthorizedIntegResourceRepo{IntegID: ai.ID}, 2)
	})

	t.Run("validates data", func(t *testing.T) {
		ai, _ := prep(t)
		err := InsertAuthorizedIntegration(t.Context(), ai, nil)
		require.ErrorIs(t, err, authz.ErrSpecifiedReposNone)
	})
}
