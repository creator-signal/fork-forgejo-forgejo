// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	audit_model "forgejo.org/models/audit"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIAdminAuditEvents(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// Insert directly via the model so the dataset is deterministic regardless of
	// whether auditing is enabled (the service-level recorder is gated by config).
	require.NoError(t, audit_model.InsertEvent(db.DefaultContext, &audit_model.Event{
		Action:    audit_model.UserAccessTokenAdd,
		DoerID:    1,
		DoerName:  "user1",
		IPAddress: "192.0.2.1",
		Message:   `Created access token "integration".`,
	}))
	require.NoError(t, audit_model.InsertEvent(db.DefaultContext, &audit_model.Event{
		Action:   audit_model.UserLogin,
		DoerID:   1,
		DoerName: "user1",
		Message:  "Signed in.",
	}))

	t.Run("admin lists all events", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		token := getUserToken(t, "user1", auth_model.AccessTokenScopeReadAdmin)
		req := NewRequest(t, "GET", "/api/v1/admin/audit").AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var events []*api.AuditEvent
		DecodeJSON(t, resp, &events)
		assert.Len(t, events, 2)
		assert.Equal(t, "2", resp.Header().Get("X-Total-Count"))
	})

	t.Run("filter by action", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		token := getUserToken(t, "user1", auth_model.AccessTokenScopeReadAdmin)
		req := NewRequest(t, "GET", "/api/v1/admin/audit?action=user_access_token_add").AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var events []*api.AuditEvent
		DecodeJSON(t, resp, &events)
		require.Len(t, events, 1)
		assert.Equal(t, "user_access_token_add", events[0].Action)
		assert.Equal(t, "192.0.2.1", events[0].IPAddress)
		assert.Equal(t, "user1", events[0].DoerName)
	})

	t.Run("non-admin is forbidden", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		token := getUserToken(t, "user2", auth_model.AccessTokenScopeReadAdmin)
		req := NewRequest(t, "GET", "/api/v1/admin/audit").AddTokenAuth(token)
		MakeRequest(t, req, http.StatusForbidden)
	})
}
