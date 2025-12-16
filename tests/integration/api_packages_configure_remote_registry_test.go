// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	rr_model "forgejo.org/models/remote_registry"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"
)

func TestConfigureRemoteRegistry(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
	session := loginUser(t, user.Name)
	tokenWritePackage := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWritePackage)

	t.Run("ConfigureRemoteRegistry", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		rr := api.CreateRemoteRegistryOption{
			Name:        "testreg",
			RemoteType:  "container",
			RemoteURL:   "https://example.registry.com",
			RemoteUser:  "someUser",
			RemoteToken: "asdfwoe324lkjsdf0242523",
		}

		req := NewRequestWithJSON(t, "POST", fmt.Sprintf("/api/v1/packages/%s/remote-registry", user.Name), &rr).AddTokenAuth(tokenWritePackage)
		MakeRequest(t, req, http.StatusCreated)

		unittest.AssertExistsAndLoadBean(t, &rr_model.RemoteRegistry{
			ID:   int64(1),
			Name: rr.Name,
		})
	})
}
