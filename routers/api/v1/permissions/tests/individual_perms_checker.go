// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.IndividualPermsChecker, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"user": "IndividualPermsChecker",
			}, map[string]string{}),
		},
		{
			data: newTestData(map[string]string{
				"user": "IndividualPermsCheckerprivate",
			}, map[string]string{}),
			error: "Visit Project",
		},
		{
			data: newTestData(map[string]string{
				"user": "IndividualPermsCheckerlimited",
			}, map[string]string{
				"doer": "anonymous",
			}),
			error: "Visit Project",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetOwnDefault("user", data.GetShared("doer"))
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasOwn("user") {
			name := data.GetOwn("user")
			fixtureCreateUser(t, &user_model.User{Name: name})
			permissions.SetUser(fixtureGetUser(t, name))
		}
	},
})
