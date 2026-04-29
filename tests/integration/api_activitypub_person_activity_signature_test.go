// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestActivityPubPersonActivityRequiresSignature verifies that the
// /api/v1/activitypub/user-id/{id}/activities/{aid} and .../activity
// endpoints reject unsigned requests, matching the protection on every
// neighbouring federation endpoint (Person, PersonInbox, PersonFeed).
//
// Regression test for audit finding T8-001: prior to the fix the activity
// sub-group was registered without activitypub.ReqHTTPSignature(), allowing
// anonymous read of activity for any user (including activity referencing
// private repos).
func TestActivityPubPersonActivityRequiresSignature(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()
	defer test.MockVariableValue(&setting.Federation.SignatureEnforced, true)()
	defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

	// Both routes under the activity sub-group must reject an unsigned
	// request. ReqHTTPSignature() returns 400 (StatusBadRequest) when the
	// request carries no signature header at all, the same behaviour as
	// the neighbouring Person/Outbox routes (see TestActivityPubPerson's
	// UnsignedRequest sub-test).
	for _, path := range []string{
		"/api/v1/activitypub/user-id/2/activities/1",
		"/api/v1/activitypub/user-id/2/activities/1/activity",
	} {
		t.Run(path, func(t *testing.T) {
			req := NewRequest(t, "GET", path)
			resp := MakeRequest(t, req, http.StatusBadRequest)
			assert.Contains(t, resp.Body.String(), "request signature verification failed")
		})
	}
}
