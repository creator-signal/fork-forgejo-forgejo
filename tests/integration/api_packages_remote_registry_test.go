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
	remote_registry_model "forgejo.org/models/remote_registry"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	mock_server "forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

var rr = api.CreateRemoteRegistryOption{
	Name:           "testreg",
	RemoteType:     "container",
	RemoteURL:      "https://example.registry.com",
	RemoteUser:     "someUser",
	RemoteToken:    "asdfwoe324lkjsdf0242523",
	RemotePassword: "somePass",
	TestConnection: true,
}

func TestCreateRemoteRegistryUser(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiRR api.RemoteRegistry
	DecodeJSON(t, resp, &apiRR)

	retrieved := unittest.AssertExistsAndLoadBean(t, &remote_registry_model.RemoteRegistry{Name: rr.Name})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, remote_registry_model.RRUser, retrieved.OwnerType)

	assert.Equal(t, rr.Name, apiRR.Name)
	assert.Equal(t, remote_registry_model.RRUser.Name(), apiRR.OwnerType)
	assert.Equal(t, user2.ID, apiRR.OwnerID)
	assert.Equal(t, rr.RemoteURL, apiRR.RemoteURL)
	assert.Equal(t, rr.RemoteUser, apiRR.RemoteUser)
	assert.Equal(t, packages.TypeContainer.Name(), apiRR.RemoteType)
}

func TestCreateDuplicateFails(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr.RemoteURL = server.URL

	// Post
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
	MakeRequest(t, req, http.StatusUnprocessableEntity)
}

func TestFailsOnNonExisting(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	// Get Bearer Token
	req := NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL))
	MakeRequest(t, req, http.StatusUnauthorized)

	req = NewRequest(t, "GET", fmt.Sprintf("%sv2/token", setting.AppURL))
	resp := MakeRequest(t, req, http.StatusOK)

	tokenResponse := &TokenResponse{}
	DecodeJSON(t, resp, &tokenResponse)

	assert.NotEmpty(t, tokenResponse.Token)
	anonymousToken := fmt.Sprintf("Bearer %s", tokenResponse.Token)

	image := "myorg/test"
	manifestDigest := "sha256:4f10484d1c1bb13e3956b4de1cd42db8e0f14a75be1617b60f2de3cd59c803c6"

	url := fmt.Sprintf("%sv2/%s/remote/%s/%s", setting.AppURL, org3.Name, "testReg", image)

	req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
		AddTokenAuth(anonymousToken)
	MakeRequest(t, req, http.StatusNotFound)
}

func TestCreateUpdateGetDeleteRemoteRegistryOrg(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr.RemoteURL = server.URL

	rr2 := api.CreateRemoteRegistryOption{
		Name:           "testreg2",
		RemoteType:     "container",
		RemoteURL:      server.URL,
		RemoteUser:     "someOtherUser",
		RemoteToken:    "",
		RemotePassword: "password",
		TestConnection: true,
	}

	// Post
	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	resp := MakeRequest(t, req, http.StatusCreated)

	var apiRR api.RemoteRegistry
	DecodeJSON(t, resp, &apiRR)

	retrieved := unittest.AssertExistsAndLoadBean(t, &remote_registry_model.RemoteRegistry{Name: rr.Name})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, remote_registry_model.RROrg, retrieved.OwnerType)
	assert.Equal(t, rr.Name, apiRR.Name)
	assert.Equal(t, remote_registry_model.RROrg.Name(), apiRR.OwnerType)
	assert.Equal(t, org3.ID, apiRR.OwnerID)
	assert.Equal(t, rr.RemoteURL, apiRR.RemoteURL)
	assert.Equal(t, rr.RemoteUser, apiRR.RemoteUser)
	assert.Equal(t, packages.TypeContainer.Name(), apiRR.RemoteType)

	// PUT
	req = NewRequestWithJSON(t, "PUT", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s", org3.Name, rr.Name), &rr2).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
	retrieved = unittest.AssertExistsAndLoadBean(t, &remote_registry_model.RemoteRegistry{Name: rr2.Name})
	assert.Equal(t, packages.TypeContainer, retrieved.RemoteType)
	assert.Equal(t, rr2.RemoteUser, retrieved.RemoteUser)

	// GET
	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name)).AddTokenAuth(tokenWritePackage)
	resp = MakeRequest(t, req, http.StatusOK)

	var apiGETRRS []api.RemoteRegistry
	DecodeJSON(t, resp, &apiGETRRS)
	assert.Equal(t, rr2.Name, apiGETRRS[0].Name)

	req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s", org3.Name, rr2.Name)).AddTokenAuth(tokenWritePackage)
	resp = MakeRequest(t, req, http.StatusOK)

	var apiGETRR api.RemoteRegistry
	DecodeJSON(t, resp, &apiGETRR)
	assert.Equal(t, rr2.Name, apiGETRR.Name)

	// DELETE
	req = NewRequest(t, "DELETE", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s", org3.Name, rr2.Name)).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusOK)
	unittest.AssertNotExistsBean(t, &remote_registry_model.RemoteRegistry{Name: rr2.Name})
}

func TestTestConnectionAPIEndpoint(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr.RemoteURL = server.URL
	rr.TestConnection = false

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)

	reqTC := NewRequest(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry/%s/test", org3.Name, rr.Name)).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, reqTC, http.StatusOK)
}

func TestCreateRemoteRegistryOrgNotAllowed(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5}) // User5 is not in org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})   // User2 is in org17, can write
	user20 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20}) // User20 is in org17, can only read
	org17 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 17})

	jsonResp := &MessageResponse{}

	// User does not belong to org3 at all
	session5 := loginUser(t, user5.Name)
	tokenWritePackage5 := getTokenForLoggedInUser(t, session5, auth_model.AccessTokenScopeWritePackage)

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org3.Name), &rr).AddTokenAuth(tokenWritePackage5)
	resp := MakeRequest(t, req, http.StatusForbidden)
	DecodeJSON(t, resp, &jsonResp)
	assert.Equal(t, "user should have specific permission or be a site admin", jsonResp.Message)

	// User does belong to org17, is in team with write permissions, but is not owner
	session10 := loginUser(t, user2.Name)
	tokenWritePackage10 := getTokenForLoggedInUser(t, session10, auth_model.AccessTokenScopeWritePackage)

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org17.Name), &rr).AddTokenAuth(tokenWritePackage10)
	resp = MakeRequest(t, req, http.StatusForbidden)
	DecodeJSON(t, resp, &jsonResp)
	assert.Equal(t, "Remote Registry creation is allowed only for owners and admins.", jsonResp.Message)

	// User does belong to org17, is in team with read permissions, but is not owner
	session11 := loginUser(t, user20.Name)
	tokenWritePackage11 := getTokenForLoggedInUser(t, session11, auth_model.AccessTokenScopeWritePackage)

	req = NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", org17.Name), &rr).AddTokenAuth(tokenWritePackage11)
	resp = MakeRequest(t, req, http.StatusForbidden)
	DecodeJSON(t, resp, &jsonResp)
	assert.Equal(t, "user should have specific permission or be a site admin", jsonResp.Message)
}

func TestConnectedBasicAuth(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	session := loginUser(t, user2.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr.RemoteURL = server.URL
	rr.RemoteToken = ""

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

	rr.RemoteURL = server.URL
	rr.RemotePassword = ""

	req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user2.Name), &rr).AddTokenAuth(tokenWritePackage)
	MakeRequest(t, req, http.StatusCreated)
}

func TestRemoteRegistryPull(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer tests.PrintCurrentTest(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // user2 is admin of org3
	org3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})

	session := loginUser(t, user.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	server := mock_server.MockForgejoRegistryServer()
	defer server.Close()

	rr.RemoteURL = server.URL

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
	tag := "latest"

	indexManifestDigest := "sha256:a45c5523b043174617cac9bda33134956c14e9add96fcacca1da36278aadbaba"
	indexManifestContent := `{"manifests":[{"annotations":{"com.docker.official-images.bashbrew.arch":"amd64","org.opencontainers.image.base.digest":"sha256:346fa035ca82052ce8ec3ddb9df460b255507acdeb1dc880a8b6930e778a553c","org.opencontainers.image.base.name":"debian:trixie-slim","org.opencontainers.image.created":"2026-02-04T23:52:22Z","org.opencontainers.image.revision":"ffe72978e08c5b0dacecd604e528f6d0741a9ae5","org.opencontainers.image.source":"https:\/\/github.com\/nginx\/docker-nginx.git#ffe72978e08c5b0dacecd604e528f6d0741a9ae5:mainline\/debian","org.opencontainers.image.url":"https:\/\/hub.docker.com\/_\/nginx","org.opencontainers.image.version":"1.29.5"},"digest":"sha256:514a9c2814250e61396ef4d6125ece1a8fbb3b0964a2ab441e9f7acf0b66b8b5","mediaType":"application\/vnd.oci.image.manifest.v1+json","platform":{"architecture":"amd64","os":"linux"},"size":2290},{"annotations":{"com.docker.official-images.bashbrew.arch":"amd64","vnd.docker.reference.digest":"sha256:514a9c2814250e61396ef4d6125ece1a8fbb3b0964a2ab441e9f7acf0b66b8b5","vnd.docker.reference.type":"attestation-manifest"},"digest":"sha256:32923807439461f47e92b606f5fe670b1791b407c62a6b4648b38f7659c034be","mediaType":"application\/vnd.oci.image.manifest.v1+json","platform":{"architecture":"unknown","os":"unknown"},"size":841}],"mediaType":"application\/vnd.oci.image.index.v1+json","schemaVersion":2}`

	blobDigest := "sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4"
	blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)

	unknownDigest := "sha256:fffffffffffffffffffffffffffffffff0000000000000000000000000000000"

	url := fmt.Sprintf("%sv2/%s/remote/%s/%s", setting.AppURL, org3.Name, rr.Name, image)

	t.Run("HEAD Manifest", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(anonymousToken)
		resp = MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, manifestDigest, resp.Header().Get("docker-content-digest"))
	})

	t.Run("HEAD Index Manifest", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, indexManifestDigest)).
			AddTokenAuth(anonymousToken)
		resp = MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, indexManifestDigest, resp.Header().Get("docker-content-digest"))
	})

	t.Run("HEAD Manifest Not Found", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/manifests/%s", url, unknownDigest)).
			AddTokenAuth(anonymousToken)
		MakeRequest(t, req, http.StatusNotFound)
	})

	t.Run("GET Manifest by Tag", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/manifests/%s", url, tag)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, manifestContent, resp.Body.String())
	})

	t.Run("GET Manifest by Digest", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/manifests/%s", url, manifestDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, manifestContent, resp.Body.String())
	})

	t.Run("GET Index Manifest by Digest", func(t *testing.T) {
		req = NewRequest(t, "GET", fmt.Sprintf("%s/manifests/%s", url, indexManifestDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, indexManifestContent, resp.Body.String())
	})

	t.Run("HEAD Blob", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/blobs/%s", url, blobDigest)).
			AddTokenAuth(anonymousToken)
		resp := MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, blobDigest, resp.Header().Get("docker-content-digest"))
	})

	t.Run("HEAD Blob Not Found", func(t *testing.T) {
		req = NewRequest(t, "HEAD", fmt.Sprintf("%s/blobs/%s", url, unknownDigest)).
			AddTokenAuth(anonymousToken)
		MakeRequest(t, req, http.StatusNotFound)
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
