// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oci "github.com/opencontainers/image-spec/specs-go/v1"
)

func uploadFlatpakPackage(t *testing.T, user *user_model.User) {
	t.Helper()

	url := fmt.Sprintf("%sv2/%s/org.example.app", setting.AppURL, user.LowerName)

	blobContent, _ := base64.StdEncoding.DecodeString(`H4sIAAAJbogA/2IYBaNgFIxYAAgAAP//Lq+17wAEAAA=`)
	blobDigest := "sha256:" + sha256Hash(string(blobContent))

	var image oci.Image
	image.Config.Labels = make(map[string]string)
	image.Config.Labels["org.flatpak.ref"] = "app/org.example.App/x86_64/master"
	image.Config.Labels["org.flatpak.runtime-repo"] = "https://dl.flathub.org/repo/flathub.flatpakrepo"
	configContent, err := json.Marshal(&image)
	require.NoError(t, err)

	configDigest := "sha256:" + sha256Hash(string(configContent))

	manifestContent := fmt.Sprintf(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:%s","size":%d},"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","digest":"sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4","size":32}]}`, sha256Hash(string(configContent)), len(configContent))

	req := NewRequestWithBody(t, "POST", fmt.Sprintf("%s/blobs/uploads?digest=%s", url, blobDigest), bytes.NewReader(blobContent)).
		AddBasicAuth(user.Name)
	MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithBody(t, "POST", fmt.Sprintf("%s/blobs/uploads?digest=%s", url, configDigest), bytes.NewReader(configContent)).
		AddBasicAuth(user.Name)
	MakeRequest(t, req, http.StatusCreated)

	req = NewRequestWithBody(t, "PUT", fmt.Sprintf("%s/manifests/latest", url), strings.NewReader(manifestContent)).
		AddBasicAuth(user.Name).
		SetHeader("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	MakeRequest(t, req, http.StatusCreated)
}

func TestPackageFlatpakIndex(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	uploadFlatpakPackage(t, user)

	req := NewRequest(t, "GET", fmt.Sprintf("/api/packages/%s/container/index/static?label:org.flatpak.ref:exists=1&architecture=amd64&os=linux&tag=latest", user.LowerName))
	resp := MakeRequest(t, req, http.StatusOK)

	type indexImage struct {
		Tags         []string
		Digest       string
		MediaType    string
		OS           string
		Architecture string
		Annotations  map[string]string
		Labels       map[string]string
	}

	type indexRepository struct {
		Name   string
		Images []indexImage
	}

	type indexResponse struct {
		Registry string
		Results  []indexRepository
	}

	var index indexResponse
	DecodeJSON(t, resp, &index)

	assert.Equal(t, "app/org.example.App/x86_64/master", index.Results[0].Images[0].Labels["org.flatpak.ref"])
}

func TestPackageFlatpakFlatpakrepo(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	req := NewRequest(t, "GET", fmt.Sprintf("/api/packages/%s/container/flatpak/repo.flatpakrepo", user.LowerName))
	resp := MakeRequest(t, req, http.StatusOK)

	assert.Equal(t, "application/vnd.flatpak.repo", resp.Header()["Content-Type"][0])

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	content := string(body)

	assert.Contains(t, content, fmt.Sprintf("Title=%s-%s", setting.AppName, user.Name))
	assert.Contains(t, content, fmt.Sprintf("Url=oci+%sapi/packages/%s/container\n", setting.AppURL, user.Name))
	assert.Contains(t, content, fmt.Sprintf("Homepage=%s\n", user.HTMLURL()))
	assert.Contains(t, content, fmt.Sprintf("Comment=Flatpak repo of %s/%s\n", setting.AppName, user.Name))
	assert.Contains(t, content, fmt.Sprintf("Description=Flatpak repo of %s/%s\n", setting.AppName, user.Name))
	assert.Contains(t, content, fmt.Sprintf("Icon=%s\n", user.AvatarLink(context.Background())))
}

func TestPackageFlatpakFlatpakref(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	uploadFlatpakPackage(t, user)

	req := NewRequest(t, "GET", fmt.Sprintf("/api/packages/%s/container/flatpak/ref/org.example.app/latest/ref.flatpakref", user.LowerName))
	resp := MakeRequest(t, req, http.StatusOK)

	assert.Equal(t, "application/vnd.flatpak.ref", resp.Header()["Content-Type"][0])

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	content := string(body)

	assert.Contains(t, content, "Name=org.example.App")
	assert.Contains(t, content, fmt.Sprintf("Title=org.example.App from %s\n", setting.AppName))
	assert.Contains(t, content, fmt.Sprintf("SuggestRemoteName=%s-%s", strings.ToLower(setting.AppName), user.LowerName))
	assert.Contains(t, content, fmt.Sprintf("Url=oci+%sapi/packages/%s/container", setting.AppURL, user.LowerName))
	assert.Contains(t, content, "Branch=master")
	assert.Contains(t, content, "RuntimeRepo=https://dl.flathub.org/repo/flathub.flatpakrepo")
	assert.Contains(t, content, "IsRuntime=false")
}
