// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package audit_test

import (
	"testing"

	audit_model "forgejo.org/models/audit"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInsertAndQueryEvents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	require.EqualValues(t, 0, audit_model.CountEvents(db.DefaultContext))

	require.NoError(t, audit_model.InsertEvent(db.DefaultContext, &audit_model.Event{
		Action:   audit_model.UserLogin,
		DoerID:   1,
		DoerName: "user1",
		Message:  "Signed in.",
	}))
	require.NoError(t, audit_model.InsertEvent(db.DefaultContext, &audit_model.Event{
		Action:    audit_model.UserAccessTokenAdd,
		DoerID:    1,
		DoerName:  "user1",
		IPAddress: "192.0.2.1",
		Message:   `Created access token "test".`,
	}))

	assert.EqualValues(t, 2, audit_model.CountEvents(db.DefaultContext))

	events, err := audit_model.Events(db.DefaultContext, 1, 10)
	require.NoError(t, err)
	require.Len(t, events, 2)

	// newest first
	assert.Equal(t, audit_model.UserAccessTokenAdd, events[0].Action)
	assert.Equal(t, "192.0.2.1", events[0].IPAddress)
	assert.NotZero(t, events[0].CreatedUnix)
	assert.Equal(t, audit_model.UserLogin, events[1].Action)
}
