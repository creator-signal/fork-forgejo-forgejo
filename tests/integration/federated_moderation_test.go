// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"forgejo.org/models/forgefed"
	"forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/routers"
	"forgejo.org/services/federation"
	"forgejo.org/tests"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fastjson"
)

func TestFederatedModeration(t *testing.T) {
	defer unittest.OverrideFixtures("tests/integration/fixtures/FederatedModerationTests")()
	defer test.MockVariableValue(&setting.Moderation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()
	federation.Init()

	locale := translation.NewLocale("en-US")

	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.Header["Content-Type"][0], "application/ld+json")
		assert.Contains(t, r.Header["Signature"][0], "/api/v1/activitypub/actor#main-key")
		assert.Contains(t, r.Header["Signature"][0], "algorithm=\"hs2019\"")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		jsonVal, err := fastjson.ParseBytes(body)
		require.NoError(t, err)

		activity := ap.JSONUnmarshalToItem(jsonVal)

		assert.Equal(t, ap.FlagType, activity.GetType())
		flag, ok := activity.(*ap.Flag)
		require.True(t, ok)

		assert.Contains(t, flag.ID, "/api/v1/activitypub/reports/")

		if strings.Contains(r.URL.Path, "user") {
			assert.Equal(t, "Other: User is suspected to have stolen the forgejo fish.", flag.Content.First().String())
			assert.Contains(t, flag.Object, "/_federation/user/1")
		} else {
			assert.Equal(t, "Illegal content: Unauthorized fork of the software frogejo.", flag.Content.First().String())
			assert.Contains(t, flag.Object, "/api/v1/activitypub/repository-id/1")
		}

		result := federation.NewServiceResultWithBytes(http.StatusAccepted, []byte(`{"status":"Accepted"}`))

		_, err = w.Write(result.Bytes)
		require.NoError(t, err)
	}))

	defer localServer.Close()

	t.Run("Basic_Variables_User", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()

		session := loginUser(t, "user1")

		// Federated user
		req := NewRequest(t, "GET", fmt.Sprintf("/report_abuse?type=user&id=1001"))
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		federationHostID := htmlDoc.GetInputValueByName("federation_host_id")
		activityPubID := htmlDoc.GetInputValueByName("activity_pub_id")

		moderationFieldset := htmlDoc.Find(".moderation fieldset").Text()
		federationNotice := locale.TrString("moderation.federation.notice")

		assert.Contains(t, moderationFieldset, federationNotice)
		assert.Equal(t, "1001", federationHostID)
		assert.Equal(t, "http://localhost/_federation_mock/user/1", activityPubID)

		// Local user
		req = NewRequest(t, "GET", fmt.Sprintf("/report_abuse?type=user&id=2"))
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc = NewHTMLParser(t, resp.Body)

		federationHostID = htmlDoc.GetInputValueByName("federation_host_id")
		activityPubID = htmlDoc.GetInputValueByName("activity_pub_id")

		moderationFieldset = htmlDoc.Find(".moderation fieldset").Text()
		federationNotice = locale.TrString("moderation.federation.notice")

		assert.NotContains(t, moderationFieldset, federationNotice)
		assert.Equal(t, "-1", federationHostID)
		assert.Empty(t, activityPubID)
	})

	t.Run("Basic_Variables_Repo", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()

		session := loginUser(t, "user1")

		// Federated repo
		req := NewRequest(t, "GET", fmt.Sprintf("/report_abuse?type=repo&id=1"))
		resp := session.MakeRequest(t, req, http.StatusOK)
		htmlDoc := NewHTMLParser(t, resp.Body)

		federationHostID := htmlDoc.GetInputValueByName("federation_host_id")
		activityPubID := htmlDoc.GetInputValueByName("activity_pub_id")

		moderationFieldset := htmlDoc.Find(".moderation fieldset").Text()
		federationNotice := locale.TrString("moderation.federation.notice")

		assert.Contains(t, moderationFieldset, federationNotice)
		assert.Equal(t, "1", federationHostID)
		assert.Equal(t, "https://forge.example.com/api/v1/activitypub/repository-id/1", activityPubID)

		// Local repo
		req = NewRequest(t, "GET", "/report_abuse?type=repo&id=42")
		resp = session.MakeRequest(t, req, http.StatusOK)
		htmlDoc = NewHTMLParser(t, resp.Body)

		federationHostID = htmlDoc.GetInputValueByName("federation_host_id")
		activityPubID = htmlDoc.GetInputValueByName("activity_pub_id")

		moderationFieldset = htmlDoc.Find(".moderation fieldset").Text()
		federationNotice = locale.TrString("moderation.federation.notice")

		assert.NotContains(t, moderationFieldset, federationNotice)
		assert.Equal(t, "-1", federationHostID)
		assert.Empty(t, activityPubID)
	})

	onApplicationRun(t, func(t *testing.T, url *url.URL) {
		session := loginUser(t, "user1")

		// Update FederationHost, FederatedUser AND FollowingRepo to match the test
		// HTTP server inbox
		ctx := t.Context()
		federatedUserAPID := "/_federation/user/1"
		federatedUser := user.FederatedUser{
			ID:                    1001,
			UserID:                1001,
			ExternalID:            fmt.Sprintf("%s%s", localServer.URL, federatedUserAPID),
			FederationHostID:      1001,
			InboxPath:             fmt.Sprintf("%s/inbox", federatedUserAPID),
			NormalizedOriginalURL: fmt.Sprintf("%s%s", localServer.URL, federatedUserAPID),
		}

		require.NoError(t, user.UpdateFederatedUser(ctx, &federatedUser))

		// FederationHost
		localServerURL, err := url.Parse(localServer.URL)
		require.NoError(t, err)

		federationHostPortStr := localServerURL.Port()
		if federationHostPortStr == "" {
			if localServerURL.Scheme == "http" {
				federationHostPortStr = "80"
			} else {
				federationHostPortStr = "443"
			}
		}

		federationHostPort, err := strconv.ParseUint(federationHostPortStr, 10, 16)
		require.NoError(t, err)

		nodeInfo := forgefed.NodeInfo{
			SoftwareName: "forgejo",
		}

		federationHost := forgefed.FederationHost{
			ID:         1001,
			HostFqdn:   localServerURL.Hostname(),
			HostPort:   uint16(federationHostPort),
			HostSchema: localServerURL.Scheme,
			NodeInfo:   nodeInfo,
		}

		require.NoError(t, forgefed.UpdateFederationHost(ctx, &federationHost))

		followingRepo := repo.FollowingRepo{
			ID:               1,
			RepoID:           1,
			FederationHostID: 1,
			ExternalID:       "1",
			URI:              fmt.Sprintf("%s/api/v1/activitypub/repository-id/1", localServer.URL),
		}

		require.NoError(t, repo.UpdateFollowingRepo(ctx, followingRepo))

		// Send an abuse report to a user
		reportURL := fmt.Sprintf("%sreport_abuse", url)
		req := NewRequestWithValues(t, "POST", reportURL, map[string]string{
			"content_id":         "1001",
			"content_type":       "1",
			"federation_host_id": "1001",
			"activity_pub_id":    federatedUser.ExternalID,
			"abuse_category":     "1",
			"remarks":            "User is suspected to have stolen the forgejo fish.",
			"forward_remote":     "on",
		})

		session.MakeRequest(t, req, http.StatusSeeOther)

		// Send an abuse report to a repository
		req = NewRequestWithValues(t, "POST", reportURL, map[string]string{
			"content_id":         "1",
			"content_type":       "2",
			"federation_host_id": "1",
			"activity_pub_id":    federatedUser.ExternalID,
			"abuse_category":     "4",
			"remarks":            "Unauthorized fork of the software frogejo.",
			"forward_remote":     "on",
		})

		session.MakeRequest(t, req, http.StatusSeeOther)
	})
}
