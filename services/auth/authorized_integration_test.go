// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"strings"
	"testing"

	"forgejo.org/models/auth"
	"forgejo.org/services/authz"

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
