// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestCreateRemoteRegistryUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	rr := api.CreateRemoteRegistryOption{
		Name:        "testreg",
		RemoteType:  "container",
		RemoteURL:   "https://example.registry.com",
		RemoteUser:  "someUser",
		RemoteToken: "asdfwoe324lkjsdf0242523",
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)

	retrieved := unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{Name: "testreg"})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr_model.RROrg, retrieved.OwnerType)
}

func TestCreateRemoteRegistryOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	rr := api.CreateRemoteRegistryOption{
		Name:        "testreg",
		RemoteType:  "container",
		RemoteURL:   "https://example.registry.com",
		RemoteUser:  "someUser",
		RemoteToken: "asdfwoe324lkjsdf0242523",
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", repo1.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)

	retrieved := unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{Name: "testreg"})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr_model.RRRepo, retrieved.OwnerType)
}
