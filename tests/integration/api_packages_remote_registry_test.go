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
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	mock_server "forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestCreateRemoteRegistryUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	rr := api.CreateRemoteRegistryOption{
		Name:        "testreg",
		RemoteType:  "container",
		RemoteURL:   "https://example.registry.com",
		RemoteUser:  "someUser",
		RemoteToken: "asdfwoe324lkjsdf0242523",
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiRR api.RemoteRegistry
	DecodeJSON(t, resp, &apiRR)

	retrieved := unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{Name: "testreg"})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr_model.RRUser, retrieved.OwnerType)

	assert.Equal(t, rr.Name, apiRR.Name)
	assert.Equal(t, rr_model.RRUser.Name(), apiRR.OwnerType)
	assert.Equal(t, user2.ID, apiRR.OwnerID)
	assert.Equal(t, rr.RemoteURL, apiRR.RemoteURL)
	assert.Equal(t, rr.RemoteUser, apiRR.RemoteUser)
	assert.Equal(t, packages.TypeContainer.Name(), apiRR.RemoteType)
}

func TestCreateRemoteRegistryOrg(t *testing.T) {
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
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiRR api.RemoteRegistry
	DecodeJSON(t, resp, &apiRR)

	retrieved := unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{Name: "testreg"})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr_model.RROrg, retrieved.OwnerType)

	assert.Equal(t, rr.Name, apiRR.Name)
	assert.Equal(t, rr_model.RROrg.Name(), apiRR.OwnerType)
	assert.Equal(t, org3.ID, apiRR.OwnerID)
	assert.Equal(t, rr.RemoteURL, apiRR.RemoteURL)
	assert.Equal(t, rr.RemoteUser, apiRR.RemoteUser)
	assert.Equal(t, packages.TypeContainer.Name(), apiRR.RemoteType)
}

func TestTestConnectionAPIEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockRegistryServer()
	defer server.Close()

	rr := api.CreateRemoteRegistryOption{
		Name:        "testreg",
		RemoteType:  "container",
		RemoteURL:   server.URL,
		RemoteUser:  "someUser",
		RemoteToken: "asdfwoe324lkjsdf0242523",
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)

	reqTC := NewRequest(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s/test", org3.Name, rr.Name)).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, reqTC, http.StatusOK)
}

func TestCreateRemoteRegistryOrgNotAllowed(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user5.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	rr := api.CreateRemoteRegistryOption{
		Name:        "testreg",
		RemoteType:  "container",
		RemoteURL:   "https://example.registry.com",
		RemoteUser:  "someUser",
		RemoteToken: "asdfwoe324lkjsdf0242523",
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusForbidden)
}

func TestConnectedBasicAuth(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockRegistryServer()
	defer server.Close()

	rr := api.CreateRemoteRegistryOption{
		Name:           "testreg",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someUser",
		RemotePassword: "somePW",
		RemoteToken:    "",
		TestConnection: true,
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
}

func TestConnectedToken(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockRegistryServer()
	defer server.Close()

	rr := api.CreateRemoteRegistryOption{
		Name:           "testreg",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someUser",
		RemoteToken:    "someToken",
		TestConnection: true,
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
}
