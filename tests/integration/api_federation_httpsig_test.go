// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/services/contexttest"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFederationHttpSigValidation(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		userID := 2
		userURL := fmt.Sprintf("%sapi/v1/activitypub/user-id/%d", u, userID)

		user1 := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1})

		ctx, _ := contexttest.MockAPIContext(t, userURL)
		clientFactory, err := activitypub.NewClientFactoryWithTimeout(60 * time.Second)
		require.NoError(t, err)

		apClient, err := clientFactory.WithKeys(ctx, user1, user1.KeyID())
		require.NoError(t, err)

		// HACK HACK HACK: the host part of the URL gets set to which IP forgejo is
		// listening on, NOT localhost, which is the Domain given to forgejo which
		// is then used for eg. the keyID all requests
		applicationKeyID := fmt.Sprintf("%sapi/v1/activitypub/actor#main-key", setting.AppURL)
		actorKeyID := fmt.Sprintf("%sapi/v1/activitypub/user-id/1#main-key", setting.AppURL)

		// Unsigned request
		t.Run("UnsignedRequest", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Check for missing public keys
		t.Run("ValidateEmptyCaches", func(t *testing.T) {
			_, err := forgefed.FindFederationHostByKeyID(db.DefaultContext, applicationKeyID)
			require.Error(t, err)
			assert.True(t, forgefed.IsErrFederationHostNotFound(err))

			_, _, err = user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.Error(t, err)
			assert.True(t, user.IsErrFederatedUserNotExists(err))
		})

		// Signed request
		t.Run("SignedRequest", func(t *testing.T) {
			resp, err := apClient.Get(userURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// Check for cached public keys
		t.Run("ValidateCaches", func(t *testing.T) {
			host, err := forgefed.FindFederationHostByKeyID(db.DefaultContext, applicationKeyID)
			require.NoError(t, err)
			assert.NotNil(t, host)
			assert.True(t, host.PublicKey.Valid)

			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			assert.NotNil(t, user)
			assert.True(t, user.PublicKey.Valid)
		})

		// Disable signature validation
		defer test.MockVariableValue(&setting.Federation.SignatureEnforced, false)()

		// Unsigned request
		t.Run("SignatureValidationDisabled", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusOK)
		})
	})
}

// TestFederationActivityPubRouteSignatureCoverage walks every route under
// /api/v1/activitypub and pins the signature-gate behavior. The test
// introspects the live router and asserts the registered route set equals
// the table below: a new route added to api.go without a matching entry
// here fails the coverage check, forcing the author to make an explicit
// requireSig decision.
func TestFederationActivityPubRouteSignatureCoverage(t *testing.T) {
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user1 := unittest.AssertExistsAndLoadBean(t, &user.User{ID: 1})
		ctx, _ := contexttest.MockAPIContext(t, u.String())
		clientFactory, err := activitypub.NewClientFactoryWithTimeout(60 * time.Second)
		require.NoError(t, err)
		apClient, err := clientFactory.WithKeys(ctx, user1, user1.KeyID())
		require.NoError(t, err)

		type route struct {
			name       string
			method     string
			pattern    string // chi pattern under /api/v1/activitypub, e.g. /user-id/{user-id}
			concrete   string // path used to issue the test request
			requireSig bool
		}
		routes := []route{
			{"Person", "GET", "/user-id/{user-id}", "/user-id/2", true},
			{"PersonInbox", "POST", "/user-id/{user-id}/inbox", "/user-id/2/inbox", true},
			{"PersonActivityNote", "GET", "/user-id/{user-id}/activities/{activity-id}", "/user-id/2/activities/1", true},
			{"PersonActivity", "GET", "/user-id/{user-id}/activities/{activity-id}/activity", "/user-id/2/activities/1/activity", true},
			{"PersonOutbox", "GET", "/user-id/{user-id}/outbox", "/user-id/2/outbox", true},
			// requireSig is false only for the bare /actor route — that endpoint
			// hosts the public key peers need to verify our signed requests, so
			// gating it would break the HTTP-signature bootstrap
			// (services/federation/signature_service.go fetchKeyFromAp).
			{"Actor", "GET", "/actor", "/actor", false},
			{"ActorInbox", "POST", "/actor/inbox", "/actor/inbox", true},
			{"ActorOutbox", "GET", "/actor/outbox", "/actor/outbox", true},
			{"Repository", "GET", "/repository-id/{repository-id}", "/repository-id/2", true},
			{"RepositoryInbox", "POST", "/repository-id/{repository-id}/inbox", "/repository-id/2/inbox", true},
			{"RepositoryOutbox", "GET", "/repository-id/{repository-id}/outbox", "/repository-id/2/outbox", true},
		}

		t.Run("RouteSetMatchesRouter", func(t *testing.T) {
			const prefix = "/api/v1/activitypub"
			expected := map[string]bool{}
			for _, r := range routes {
				expected[r.method+" "+prefix+r.pattern] = true
			}
			actual := map[string]bool{}
			err := chi.Walk(testWebRoutes.R, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
				if strings.HasPrefix(route, prefix+"/") || route == prefix {
					actual[method+" "+route] = true
				}
				return nil
			})
			require.NoError(t, err)
			assert.Equal(t, expected, actual,
				"activitypub route set drifted from this test's table — add or remove an entry above and pick requireSig deliberately")
		})

		const sigErrMsg = "request signature verification failed"

		for _, r := range routes {
			full := fmt.Sprintf("%sapi/v1/activitypub%s", u, r.concrete)

			t.Run(r.name+"_Unsigned", func(t *testing.T) {
				req := NewRequest(t, r.method, full)
				if r.requireSig {
					resp := MakeRequest(t, req, http.StatusBadRequest)
					assert.Contains(t, resp.Body.String(), sigErrMsg)
				} else {
					MakeRequest(t, req, http.StatusOK)
				}
			})

			t.Run(r.name+"_Signed", func(t *testing.T) {
				var resp *http.Response
				var err error
				switch r.method {
				case "GET":
					resp, err = apClient.Get(full)
				case "POST":
					resp, err = apClient.Post([]byte("{}"), full)
				default:
					t.Fatalf("unhandled method %q", r.method)
				}
				require.NoError(t, err)
				defer resp.Body.Close()
				body, _ := io.ReadAll(resp.Body)
				assert.NotContains(t, string(body), sigErrMsg,
					"signed %s %s rejected by signature gate (status %d): %s",
					r.method, r.concrete, resp.StatusCode, string(body))
			})
		}
	})
}
