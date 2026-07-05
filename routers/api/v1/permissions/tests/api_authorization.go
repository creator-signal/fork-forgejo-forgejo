// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.APIAuthorization, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer": "anonymous",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer": "doerregular",
			}),
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("doer", "doerregular")
		if data.GetShared("doer") == user_model.ActionsUserName {
			data.SetSharedDefault("repository", "userowner/repositorypublic")
		}
		data.SetSharedDefault("scope", "read:repository")
		data.SetSharedDefault("level", "read")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasShared("repository") && data.GetShared("doer") == user_model.ActionsUserName {
			fixtureSetRepository(t, permissions, data.GetShared("repository"), data.GetShared("repository-init"))
		}
		fixtureSetDoer(t, permissions, data)
	},
})
