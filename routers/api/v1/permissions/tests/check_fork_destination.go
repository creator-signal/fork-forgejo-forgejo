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
			data: newTestData(map[string]string{
				"forkOrg":      "someorg1",
				"forkOrgOwner": "someorgowner",
			}, newSharedData().
				SetDoer("someorgowner").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":              "someorg1",
				"forkOrgOwner":         "someorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "true",
			}, newSharedData().
				SetDoer("someuser").
				SetRepositoryName("someuser/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":              "someorg1",
				"forkOrgOwner":         "someorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "false",
			}, newSharedData().
				SetDoer("someuser").
				SetRepositoryName("someuser/repositorypublic"),
			),
			error: "User is not allowed to create repos in Organisation",
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":      "someorg2",
				"forkOrgOwner": "someorgowner",
			}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "User is no Member of Organisation 'someorg2'",
		},
		{
			data: newTestData(map[string]string{
				"forkOrg": "unknownOrg",
			}, newSharedData().
				SetDoer("someorgowner").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "org does not exist",
		},
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		require.True(t, data.Has("forkOrg"))
		if data.Get("forkOrg") == "unknownOrg" {
			return
		}
		require.True(t, data.Has("forkOrgOwner"))
		name := data.Get("forkOrg")
		owner := data.Get("forkOrgOwner")
		org := fixtureCreateOrg(t, &org_model.Organization{Name: name}, &user_model.User{Name: owner})

		if data.Has("team") {
			fixtureCreateTeam(t, org, data.shared.Doer(), &forgery.CreateTeamOptions{
				Name:             data.Get("team"),
				CanCreateOrgRepo: data.Get("teamCanCreateOrgRepo") != "false",

				Mode: perm.AccessModeWrite,
			})
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		forkOrg := data.Get("forkOrg")
		t.Logf("calling CheckForkDestination(ctx, %s)", forkOrg)
		apiv1_permissions.CheckForkDestination(ctx, &forkOrg)
	},
})
