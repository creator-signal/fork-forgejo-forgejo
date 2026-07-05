// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.RepoAccess, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "anonymous",
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "doeradmin",
				"repository": "userowner/repositoryprivate",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositoryprivate",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "anonymous",
				"repository": "userowner/repositoryprivate",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       user_model.ActionsUserName,
				"repository": "userowner/repositorypublic",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":        user_model.ActionsUserName,
				"task.RepoID": "unrelated",
				"repository":  "userowner/repositorypublic",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":                   user_model.ActionsUserName,
				"task.IsForkPullRequest": "true",
				"repository":             "userowner/repositorypublic",
			}),
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("repository", "userowner/repositorypublic")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureSetRepository(t, permissions, data)
	},
})
