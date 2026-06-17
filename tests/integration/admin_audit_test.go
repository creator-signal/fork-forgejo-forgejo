// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	audit_model "forgejo.org/models/audit"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminAuditLog(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	defer test.MockVariableValue(&setting.Audit.Enabled, true)()

	doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	require.NoError(t, audit_model.InsertEvent(db.DefaultContext, &audit_model.Event{
		Action:    audit_model.UserAccessTokenAdd,
		DoerID:    doer.ID,
		DoerName:  doer.Name,
		IPAddress: "192.0.2.1",
		Message:   `Created access token "integration".`,
	}))

	session := loginUser(t, "user1")
	resp := session.MakeRequest(t, NewRequest(t, "GET", "/admin/audit"), http.StatusOK)

	body := resp.Body.String()
	assert.Contains(t, body, "user_access_token_add")
	assert.Contains(t, body, "Created access token")
	assert.Contains(t, body, "192.0.2.1")
}
