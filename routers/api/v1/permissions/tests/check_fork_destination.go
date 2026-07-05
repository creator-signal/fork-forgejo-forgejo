// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	org_model "forgejo.org/models/organization"
	"forgejo.org/models/perm"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.CheckForkDestination, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":         "regularorgowner",
				"repository":   "userowner/repositorypublic",
				"forkOrg":      "regularorg1",
				"forkOrgOwner": "regularorgowner",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":                 "regularuser",
				"repository":           "regularuser/repositorypublic",
				"forkOrg":              "regularorg1",
				"forkOrgOwner":         "regularorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "true",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":                 "regularuser",
				"repository":           "regularuser/repositorypublic",
				"forkOrg":              "regularorg1",
				"forkOrgOwner":         "regularorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "false",
			}),
			error: "User is not allowed to create repos in Organisation",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":         "doerregular",
				"repository":   "userowner/repositorypublic",
				"forkOrg":      "regularorg2",
				"forkOrgOwner": "regularorgowner",
			}),
			error: "User is no Member of Organisation 'regularorg2'",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":       "regularorgowner",
				"repository": "userowner/repositorypublic",
				"forkOrg":    "unknownOrg",
			}),
			error: "org does not exist",
		},
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		require.True(t, data.HasShared("forkOrg"))
		if data.GetShared("forkOrg") == "unknownOrg" {
			return
		}
		require.True(t, data.HasShared("forkOrgOwner"))
		name := data.GetShared("forkOrg")
		owner := data.GetShared("forkOrgOwner")
		org := fixtureCreateOrg(t, &org_model.Organization{Name: name}, &user_model.User{Name: owner})

		if data.HasShared("team") {
			fixtureCreateTeam(t, org, data.GetShared("doer"), &forgery.CreateTeamOptions{
				Name:             data.GetShared("team"),
				CanCreateOrgRepo: data.GetShared("teamCanCreateOrgRepo") != "false",

				Mode: perm.AccessModeWrite,
			})
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		forkOrg := data.GetShared("forkOrg")
		t.Logf("calling CheckForkDestination(ctx, %s)", forkOrg)
		apiv1_permissions.CheckForkDestination(ctx, &forkOrg)
	},
})
