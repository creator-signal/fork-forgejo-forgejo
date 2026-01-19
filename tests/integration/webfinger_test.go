// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/webfinger"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebfinger(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})

	appURL, _ := url.Parse(setting.AppURL)

	session := loginUser(t, "user1")

	ctx := t.Context()

	req := NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", user.LowerName, appURL.Host))
	resp := MakeRequest(t, req, http.StatusOK)
	assert.Equal(t, "application/jrd+json", resp.Header().Get("Content-Type"))

	var jrd webfinger.JRD
	DecodeJSON(t, resp, &jrd)
	assert.Equal(t, "acct:user2@"+appURL.Host, jrd.Subject)
	assert.ElementsMatch(t, []*webfinger.Link{
		{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: user.HTMLURL(),
		},
		{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: user.AvatarLink(ctx),
		},
		{
			Rel:  "self",
			Type: "application/activity+json",
			Href: appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(user.ID),
		},
		{
			Rel:  "http://openid.net/specs/connect/1.0/issuer",
			Href: strings.TrimSuffix(appURL.String(), "/"),
		},
	}, jrd.Links)
	assert.ElementsMatch(t, []string{user.HTMLURL(), appURL.String() + "api/v1/activitypub/user-id/" + fmt.Sprint(user.ID)}, jrd.Aliases)

	instanceReq := NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:ghost@%s", appURL.Host))
	instanceResp := MakeRequest(t, instanceReq, http.StatusOK)
	assert.Equal(t, "application/jrd+json", instanceResp.Header().Get("Content-Type"))

	var instanceActor webfinger.JRD
	DecodeJSON(t, instanceResp, &instanceActor)
	assert.Equal(t, "acct:ghost@"+appURL.Host, instanceActor.Subject)
	assert.ElementsMatch(t, []string{appURL.String() + "api/v1/activitypub/actor"}, instanceActor.Aliases)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", user.LowerName, "unknown.host"))
	MakeRequest(t, req, http.StatusBadRequest)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", "user31", appURL.Host))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s", "user31", appURL.Host))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=mailto:%s", user.Email))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=https://%s/%s/", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=https://%s/%s", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s/%s/foo", appURL.Host, user.Name))
	session.MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s", appURL.Host))
	MakeRequest(t, req, http.StatusNotFound)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=http://%s/%s/foo", "example.com", user.Name))
	MakeRequest(t, req, http.StatusBadRequest)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:%s@%s@%s", repo2.Name, repo2.OwnerName, appURL.Host))
	session.MakeRequest(t, req, http.StatusOK)

	req = NewRequest(t, "GET", fmt.Sprintf("/.well-known/webfinger?resource=acct:@%s@%s@%s", repo2.Name, repo2.OwnerName, appURL.Host))
	session.MakeRequest(t, req, http.StatusOK)
}

func TestQueryWebfinger(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, _ *url.URL) {
		defer test.MockVariableValue(&setting.Federation.Enabled, true)()
		defer test.MockVariableValue(&setting.IsProd, false)()

		appURL, _ := url.Parse(setting.AppURL)
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		ctx := t.Context()

		jrd, err := webfinger.Query(ctx, fmt.Sprintf("@%s@%s", user.LowerName, appURL.Host))
		require.NoError(t, err)

		aliases := []string{
			fmt.Sprintf("http://%s/%s", appURL.Host, user.LowerName),
			fmt.Sprintf("http://%s/api/v1/activitypub/user-id/%d", appURL.Host, user.ID),
		}

		assert.Equal(t, fmt.Sprintf("acct:%s@%s", user.LowerName, appURL.Host), jrd.Subject)
		assert.Equal(t, aliases, jrd.Aliases)
		assert.Len(t, jrd.Links, 4)

		profileActivity, err := jrd.GetProfileActivity()
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("http://%s/api/v1/activitypub/user-id/%d", appURL.Host, user.ID), profileActivity.ActivityLocation.String())

		assert.True(t, profileActivity.ProfilePage.Has())
		assert.Equal(t, fmt.Sprintf("http://%s/%s", appURL.Host, user.LowerName), profileActivity.ProfilePage.Value().String())
	})
}

func TestWebfingerTimeout(t *testing.T) {
	defer test.MockVariableValue(&setting.IsProd, false)()

	mock := test.NewFederationServerMock()
	server := mock.DistantServer(t)
	url, _ := url.Parse(server.URL)

	defer server.Close()

	ctx := t.Context()
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	// User is set to sleep for five seconds before completing
	_, err := webfinger.Query(timeoutCtx, fmt.Sprintf("@sloth@%s", url.Host))

	require.Error(t, err)
	assert.ErrorContains(t, err, context.DeadlineExceeded.Error())
}
