// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
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

func TestConnected(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := MockRegistryServer()
	defer server.Close()

	rr := api.CreateRemoteRegistryOption{
		Name:           "testreg",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someUser",
		RemoteToken:    "",
		TestConnection: true,
	}
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
}

func TestConnectedBasicAuth(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := MockRegistryServer()
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

	server := MockRegistryServer()
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

func MockRegistryServer() *httptest.Server {
	registryRoute := http.NewServeMux()

	srv := httptest.NewUnstartedServer(registryRoute)
	addr := srv.Listener.Addr()

	registryRoute.HandleFunc("/v2/",
		func(res http.ResponseWriter, req *http.Request) {
			authHeader := req.Header.Get("Authorization")
			if strings.Contains(authHeader, "Bearer") {
				res.WriteHeader(http.StatusOK)
			} else {
				res.Header().Add("docker-distribution-api-version", "registry/2.0")
				headerVal := "Bearer realm=" + "\"http://" + addr.String() + "/token\"" + ",service='registry.example.com'"
				res.Header().Add("www-authenticate", headerVal)
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	registryRoute.HandleFunc("/token",
		func(res http.ResponseWriter, req *http.Request) {
			if req.Header.Get("Authorization") != "" {
				res.WriteHeader(http.StatusOK)
			} else {
				res.WriteHeader(http.StatusUnauthorized)
			}
		})

	srv.Start()
	return srv
}
