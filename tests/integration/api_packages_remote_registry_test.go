// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/packages"
	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
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

func TestCreateUpdateDeleteRemoteRegistryOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr := api.CreateRemoteRegistryOption{
		Name:           "testreg",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someUser",
		RemoteToken:    "asdfwoe324lkjsdf0242523",
		RemotePassword: "bla",
		TestConnection: true,
	}

	rr2 := api.CreateRemoteRegistryOption{
		Name:           "testreg2",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someOtherUser",
		RemoteToken:    "",
		RemotePassword: "password",
		TestConnection: true,
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

	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s", org3.Name, rr.Name), &rr2).AddTokenAuth(tokenWritePackage)
	resp = MakeRequest(t, req, http.StatusCreated)
	retrieved = unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{Name: "testreg2"})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr2.RemoteUser, retrieved.RemoteUser)

	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s", org3.Name, rr2.Name)).AddTokenAuth(tokenWritePackage)
	resp = MakeRequest(t, req, http.StatusOK)
	unittest.AssertNotExistsBean(t, &rr_model.RemoteRegistry{Name: "testreg2"})
}

func TestTestConnectionAPIEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
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

	server := mock_server.MockForgejoRegistryServer()
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

	server := mock_server.MockForgejoRegistryServer()
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

func TestRemoteRegistryPull(t *testing.T) {
	type TokenResponse struct {
		Token string `json:"token"`
	}

	defer tests.PrepareTestEnv(t)()
	defer tests.PrintCurrentTest(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
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

	// Get Bearer Token
	req = NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL))
	MakeRequest(t, req, http.StatusUnauthorized)

	req = NewRequest(t, "GET", fmt.Sprintf("%sv2/token", setting.AppURL))
	resp := MakeRequest(t, req, http.StatusOK)

	tokenResponse := &TokenResponse{}
	DecodeJSON(t, resp, &tokenResponse)

	assert.NotEmpty(t, tokenResponse.Token)
	anonymousToken := fmt.Sprintf("Bearer %s", tokenResponse.Token)

	image := "myorg/test"
	manifestDigest := "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6"
	manifestContent := `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:4607e093bec406eaadb6f3a340f63400c9d3a7038680744c406903766b938f0d","size":1069},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`

	blobDigest := "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)

	url := fmt.Sprintf("%sv2/%s/remote/%s/%s", setting.AppURL, org3.Name, rr.Name, image)

	t.Run("HEAD Manifest", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(anonymousToken)
		MakeRequest(t, req, http.StatusOK)
	})

	t.Run("GET Manifest", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, manifestContent, resp.Body.String())
	})

	t.Run("HEAD Blob", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/blobs/%s", url, blobDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, blobDigest, resp.Header().Get("docker-content-digest"))
	})

	t.Run("GET Blob", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/blobs/%s", url, blobDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, string(blobContent), resp.Body.String())
	})

	t.Run("List Tags", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/tags/list", url)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Contains(t, resp.Body.String(), "latest")
	})

}
