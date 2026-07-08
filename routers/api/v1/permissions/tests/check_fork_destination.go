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
				"forkOrg":      "regularorg1",
				"forkOrgOwner": "regularorgowner",
			}, newSharedData().
				SetDoerName("regularorgowner").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":              "regularorg1",
				"forkOrgOwner":         "regularorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "true",
			}, newSharedData().
				SetDoerName("regularuser").
				SetRepositoryName("regularuser/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":              "regularorg1",
				"forkOrgOwner":         "regularorgowner",
				"team":                 "team1",
				"teamCanCreateOrgRepo": "false",
			}, newSharedData().
				SetDoerName("regularuser").
				SetRepositoryName("regularuser/repositorypublic"),
			),
			error: "User is not allowed to create repos in Organisation",
		},
		{
			data: newTestData(map[string]string{
				"forkOrg":      "regularorg2",
				"forkOrgOwner": "regularorgowner",
			}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "User is no Member of Organisation 'regularorg2'",
		},
		{
			data: newTestData(map[string]string{
				"forkOrg": "unknownOrg",
			}, newSharedData().
				SetDoerName("regularorgowner").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "org does not exist",
		},
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		require.True(t, data.HasOwn("forkOrg"))
		if data.GetOwn("forkOrg") == "unknownOrg" {
			return
		}
		require.True(t, data.HasOwn("forkOrgOwner"))
		name := data.GetOwn("forkOrg")
		owner := data.GetOwn("forkOrgOwner")
		org := fixtureCreateOrg(t, &org_model.Organization{Name: name}, &user_model.User{Name: owner})

		if data.HasOwn("team") {
			fixtureCreateTeam(t, org, data.shared.DoerName(), &forgery.CreateTeamOptions{
				Name:             data.GetOwn("team"),
				CanCreateOrgRepo: data.GetOwn("teamCanCreateOrgRepo") != "false",

				Mode: perm.AccessModeWrite,
			})
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		forkOrg := data.GetOwn("forkOrg")
		t.Logf("calling CheckForkDestination(ctx, %s)", forkOrg)
		apiv1_permissions.CheckForkDestination(ctx, &forkOrg)
	},
})
