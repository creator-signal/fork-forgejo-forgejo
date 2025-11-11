// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/perm"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
)

func TestAPIRepoCollaboratorPermission(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		repo2Owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo2.OwnerID})

		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
		user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
		user10 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10})
		user11 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 11})

		testCtx := NewAPITestContext(t, repo2Owner.Name, repo2.Name, auth_model.AccessTokenScopeWriteRepository)

		t.Run("RepoOwnerShouldBeOwner", func(t *testing.T) {
			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, repo2Owner.Name).
				AddTokenAuth(testCtx.Token)
			resp := MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "owner", repoPermission.Permission)
		})

		t.Run("CollaboratorWithReadAccess", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithReadAccess", doAPIAddCollaborator(testCtx, user4.Name, perm.AccessModeRead))

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user4.Name).
				AddTokenAuth(testCtx.Token)
			resp := MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "read", repoPermission.Permission)

			t.Run("CollaboratorCanReadTheirPermission", func(t *testing.T) {
				session := loginUser(t, user4.Name)
				token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)

				req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user4.Name).
					AddTokenAuth(token)
				resp := MakeRequest(t, req, http.StatusOK)

				var repoPermission api.RepoCollaboratorPermission
				DecodeJSON(t, resp, &repoPermission)

				assert.Equal(t, "read", repoPermission.Permission)
			})
		})

		t.Run("CollaboratorWithWriteAccess", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithWriteAccess", doAPIAddCollaborator(testCtx, user4.Name, perm.AccessModeWrite))

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user4.Name).
				AddTokenAuth(testCtx.Token)
			resp := MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "write", repoPermission.Permission)
		})

		t.Run("CollaboratorWithAdminAccess", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithAdminAccess", doAPIAddCollaborator(testCtx, user4.Name, perm.AccessModeAdmin))

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user4.Name).
				AddTokenAuth(testCtx.Token)
			resp := MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "admin", repoPermission.Permission)
		})

		t.Run("CollaboratorNotFound", func(t *testing.T) {
			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, "non-existent-user").
				AddTokenAuth(testCtx.Token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("CollaboratorCanQueryItsPermissions", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithReadAccess", doAPIAddCollaborator(testCtx, user5.Name, perm.AccessModeRead))

			_session := loginUser(t, user5.Name)
			_testCtx := NewAPITestContext(t, user5.Name, repo2.Name, auth_model.AccessTokenScopeReadRepository)

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user5.Name).
				AddTokenAuth(_testCtx.Token)
			resp := _session.MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "read", repoPermission.Permission)
		})

		t.Run("CollaboratorCanQueryItsPermissions", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithReadAccess", doAPIAddCollaborator(testCtx, user5.Name, perm.AccessModeRead))

			_session := loginUser(t, user5.Name)
			_testCtx := NewAPITestContext(t, user5.Name, repo2.Name, auth_model.AccessTokenScopeReadRepository)

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user5.Name).
				AddTokenAuth(_testCtx.Token)
			resp := _session.MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "read", repoPermission.Permission)
		})

		t.Run("RepoAdminCanQueryACollaboratorsPermissions", func(t *testing.T) {
			t.Run("AddUserAsCollaboratorWithAdminAccess", doAPIAddCollaborator(testCtx, user10.Name, perm.AccessModeAdmin))
			t.Run("AddUserAsCollaboratorWithReadAccess", doAPIAddCollaborator(testCtx, user11.Name, perm.AccessModeRead))

			_session := loginUser(t, user10.Name)
			_testCtx := NewAPITestContext(t, user10.Name, repo2.Name, auth_model.AccessTokenScopeReadRepository)

			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s/permission", repo2Owner.Name, repo2.Name, user11.Name).
				AddTokenAuth(_testCtx.Token)
			resp := _session.MakeRequest(t, req, http.StatusOK)

			var repoPermission api.RepoCollaboratorPermission
			DecodeJSON(t, resp, &repoPermission)

			assert.Equal(t, "read", repoPermission.Permission)
		})
	})
}

func TestAPIRepoCollaboratorOrgRepoWithSettingEnabled(t *testing.T) {
	// Test that individual collaborators are blocked on organization repositories
	// when the DISABLE_COLLABORATORS_FOR_ORGANIZATIONS setting is true.
	// This verifies that the API returns 403 Forbidden when attempting to add/delete collaborators on org repos.
	defer test.MockVariableValue(&setting.Repository.DisableCollaboratorsForOrganizations, true)()

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
		assert.Equal(t, int64(3), repo3.OwnerID) // org3
		org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})
		assert.True(t, org3.IsOrganization())

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})

		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		t.Run("AddCollaboratorToOrgRepoShouldFail", func(t *testing.T) {
			addCollaboratorOption := &api.AddCollaboratorOption{
				Permission: new(string),
			}
			*addCollaboratorOption.Permission = "write"

			req := NewRequestWithJSON(t, "PUT", "/api/v1/repos/"+org3.Name+"/"+repo3.Name+"/collaborators/"+user4.Name, addCollaboratorOption).
				AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusForbidden)

			var errMap map[string]any
			DecodeJSON(t, resp, &errMap)
			assert.Contains(t, errMap["message"], "individual collaborators are not allowed for organization repositories")
		})

		t.Run("DeleteCollaboratorFromOrgRepoShouldFail", func(t *testing.T) {
			req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/collaborators/%s", org3.Name, repo3.Name, user4.Name).
				AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusForbidden)

			var errMap map[string]any
			DecodeJSON(t, resp, &errMap)
			assert.Contains(t, errMap["message"], "individual collaborators are not allowed for organization repositories")
		})
	})
}

func TestAPIRepoCollaboratorUserRepoNotAffected(t *testing.T) {
	// Test that individual collaborators can still be added to user-owned repositories
	// even when DISABLE_COLLABORATORS_FOR_ORGANIZATIONS is true.
	// This verifies that the setting only affects organization-owned repos, not user-owned repos.
	defer test.MockVariableValue(&setting.Repository.DisableCollaboratorsForOrganizations, true)()

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo2.OwnerID})
		assert.False(t, user2.IsOrganization())

		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})

		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		t.Run("AddCollaboratorToUserRepoShouldSucceed", func(t *testing.T) {
			addCollaboratorOption := &api.AddCollaboratorOption{
				Permission: new(string),
			}
			*addCollaboratorOption.Permission = "read"

			req := NewRequestWithJSON(t, "PUT", "/api/v1/repos/"+user2.Name+"/"+repo2.Name+"/collaborators/"+user4.Name, addCollaboratorOption).
				AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)

			req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s", user2.Name, repo2.Name, user4.Name).
				AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)
		})

		t.Run("DeleteCollaboratorFromUserRepoShouldSucceed", func(t *testing.T) {
			req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/collaborators/%s", user2.Name, repo2.Name, user4.Name).
				AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNoContent)

			req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/collaborators/%s", user2.Name, repo2.Name, user4.Name).
				AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})
	})
}
