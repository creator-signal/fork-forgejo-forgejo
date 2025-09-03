// Copyright 2025 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"
	"github.com/stretchr/testify/assert"
)

func TestAPIConvert(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	user20 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	repo25 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 25})

	// Get user20's token
	session := loginUser(t, user20.Name)
	token3 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	// Get user2's token
	session = loginUser(t, user2.Name)
	token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

	req := NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/convert", user20.Name, repo25.Name)).AddTokenAuth(token3)
	resp := MakeRequest(t, req, http.StatusOK)
	var repo api.Repository
	DecodeJSON(t, resp, &repo)
	assert.NotNil(t, repo)
	assert.False(t, repo.Mirror)

	repo25edited := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 5})
	assert.False(t, repo25edited.IsMirror)

	// Test editing a non-existing repo
	name := "repodoesnotexist"
	req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/convert", user20.Name, name)).AddTokenAuth(token3)
	_ = MakeRequest(t, req, http.StatusNotFound)

	// Test copnverting by user2 who isn't owner
	req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/convert", user2.Name, repo25.Name)).AddTokenAuth(token2)
	MakeRequest(t, req, http.StatusNotFound)

	// Tests a repo with no token given so will fail
	req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/convert", user20.Name, repo25.Name))
	_ = MakeRequest(t, req, http.StatusForbidden)

	// Test converting a repo that is not a mirror does nothing
	req = NewRequest(t, "POST", fmt.Sprintf("/api/v1/repos/%s/%s/convert", user2.Name, repo1.Name)).AddTokenAuth(token2)
	_ = MakeRequest(t, req, http.StatusOK)
}
