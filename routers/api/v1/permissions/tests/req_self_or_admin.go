// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqSelfOrAdmin, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoerName("doeradmin").
				SetDoerAdmin(true)),
		},
		{
			data: newTestData(map[string]string{
				"user": "regularuser",
			}, newSharedData().
				SetDoerName("regularuser"),
			),
		},
		{
			data: newTestData(map[string]string{
				"user": "otheruser",
			}, newSharedData().
				SetDoerName("regularuser"),
			),
			error: "doer should be the site admin or be same as the contextUser",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"ReqSelfOrAdmin",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		if !data.shared.HasDoerName() {
			data.shared.SetDoerName("doeradmin")
			data.shared.SetDoerAdmin(true)
		}
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasOwn("user") {
			name := data.GetOwn("user")
			fixtureCreateUser(t, &user_model.User{Name: name})
			permissions.SetUser(fixtureGetUser(t, name))
		}
	},
})
