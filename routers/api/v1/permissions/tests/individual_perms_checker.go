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
			data: newTestData(map[string]string{}, map[string]string{
				"user": "IndividualPermsChecker",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"user": "IndividualPermsCheckerprivate",
			}),
			error: "Visit Project",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer": "anonymous",
				"user": "IndividualPermsCheckerlimited",
			}),
			error: "Visit Project",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("user", data.GetShared("doer"))
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasShared("user") && data.GetShared("user") != "anonymous" {
			name := data.GetShared("user")
			fixtureCreateUser(t, &user_model.User{Name: name})
			permissions.SetUser(fixtureGetUser(t, name))
		}
	},
})
