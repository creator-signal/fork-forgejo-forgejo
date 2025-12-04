// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/activitypub"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/services/contexttest"
	"forgejo.org/services/federation"

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

		// Unsigned request
		t.Run("UnsignedRequest", func(t *testing.T) {
			req := NewRequest(t, "GET", userURL)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Signed request
		t.Run("SignedRequest", func(t *testing.T) {
			resp, err := apClient.Get(userURL)
			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
		})

		// HACK HACK HACK: the host part of the URL gets set to which IP forgejo is
		// listening on, NOT localhost, which is the Domain given to forgejo which
		// is then used for eg. the keyID all requests
		applicationKeyID, err := federation_key.NewKeyID(fmt.Sprintf("%sapi/v1/activitypub/actor#main-key", setting.AppURL))
		require.NoError(t, err)
		actorKeyID, err := federation_key.NewKeyID(fmt.Sprintf("%sapi/v1/activitypub/user-id/1#main-key", setting.AppURL))
		require.NoError(t, err)

		// Valid key ID
		t.Run("ValidKeyID", func(t *testing.T) {
			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			require.NoError(t, user.ValidateKeyID(ctx, actorKeyID))
		})

		// Invalid key ID
		t.Run("InvalidKeyID", func(t *testing.T) {
			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)

			// bad actor user ID
			badActorKeyID := fmt.Sprintf("%sapi/v1/activitypub/user-id/2#main-key", setting.AppURL)
			keyID, err := federation_key.NewKeyID(badActorKeyID)
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))

			// bad host
			badHostKeyID := "http://bad.host/api/v1/activitypub/user-id/2#main-key"
			keyID, err = federation_key.NewKeyID(badHostKeyID)
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))

			// bad scheme
			keyURL, err := actorKeyID.IRI().URL()
			require.NoError(t, err)
			keyURL.Scheme = "https"

			keyID, err = federation_key.NewKeyID(keyURL.String())
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))
		})

		// Valid key ID
		t.Run("ValidKeyID", func(t *testing.T) {
			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)

			keyID, err := federation_key.NewKeyID(actorKeyID)
			require.NoError(t, err)

			require.NoError(t, user.ValidateKeyID(ctx, keyID))
		})

		// Invalid key ID
		t.Run("InvalidKeyID", func(t *testing.T) {
			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)

			// bad actor user ID
			badActorKeyID := fmt.Sprintf("%sapi/v1/activitypub/user-id/2#main-key", setting.AppURL)
			keyID, err := federation_key.NewKeyID(badActorKeyID)
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))

			// bad host
			badHostKeyID := "http://bad.host/api/v1/activitypub/user-id/2#main-key"
			keyID, err = federation_key.NewKeyID(badHostKeyID)
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))

			// bad scheme
			keyID, err = federation_key.NewKeyID(actorKeyID)
			require.NoError(t, err)

			keyURL, err := keyID.IRI().URL()
			require.NoError(t, err)
			keyURL.Scheme = "https"

			keyID, err = federation_key.NewKeyID(keyURL.String())
			require.NoError(t, err)

			require.Error(t, user.ValidateKeyID(ctx, keyID))
		})

		// Check for cached public keys
		t.Run("ValidateCaches", func(t *testing.T) {
			host, err := forgefed.FindFederationHostByKeyID(db.DefaultContext, applicationKeyID)
			require.NoError(t, err)
			assert.NotNil(t, host)

			hostKey, err := federation_key.FindFederationPublicKey(db.DefaultContext, applicationKeyID)
			require.NoError(t, err)
			assert.NotNil(t, hostKey)
			assert.Equal(t, hostKey.ActorID, host.ID)
			require.NoError(t, host.ValidateKeyID(hostKey.KeyID))

			_, user, err := user.FindFederatedUserByKeyID(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			assert.NotNil(t, user)

			userKey, err := federation_key.FindFederationPublicKey(db.DefaultContext, actorKeyID)
			require.NoError(t, err)
			assert.NotNil(t, userKey)
			assert.Equal(t, userKey.ActorID, user.ID)
			require.NoError(t, user.ValidateKeyID(ctx, userKey.KeyID))
		})

		t.Run("ValidateActorFromKeyID", func(t *testing.T) {
			_, err := federation.NewActorIDFromKeyID(ctx, actorKeyID)
			require.NoError(t, err)

			_, err = federation.NewActorIDFromKeyID(ctx, "http://bad.url/%^&")
			require.Error(t, err)
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
