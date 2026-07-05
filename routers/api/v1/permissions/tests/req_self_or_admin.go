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
			data: newTestData(map[string]string{}, map[string]string{
				"doer": "doeradmin",
			}),
		},
		{
			data: newTestData(map[string]string{
				"user": "regularuser",
			}, map[string]string{
				"doer": "regularuser",
			}),
		},
		{
			data: newTestData(map[string]string{
				"user": "otheruser",
			}, map[string]string{
				"doer": "regularuser",
			}),
			error: "doer should be the site admin or be same as the contextUser",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"ReqSelfOrAdmin",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("doer", "doeradmin")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasOwn("user") {
			name := data.GetOwn("user")
			fixtureCreateUser(t, &user_model.User{Name: name})
			permissions.SetUser(fixtureGetUser(t, name))
		}
	},
})
