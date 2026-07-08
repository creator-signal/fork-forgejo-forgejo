// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTest(apiv1_permissions.ReqTeamMembership, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"org":  "ReqTeamMembership",
				"team": org_model.OwnerTeamName,
			}, newSharedData()),
		},
		{
			data: newTestData(map[string]string{
				"org":  "ReqTeamMembership",
				"team": org_model.OwnerTeamName,
			}, newSharedData().
				SetDoerName("doeradmin").
				SetDoerAdmin(true),
			),
		},
		{
			data: newTestData(map[string]string{
				"orgOwner": "orgOwner",
				"org":      "ReqTeamMembership",
				"teams":    "team1:regularuser",
				"team":     "team1",
			}, newSharedData().
				SetDoerName("regularuser"),
			),
		},
		{
			data: newTestData(map[string]string{
				"orgOwner": "orgOwner",
				"org":      "ReqTeamMembership",
				"teams":    "team1:regularuser,team2:otheruser",
				"team":     "team2",
			}, newSharedData().
				SetDoerName("regularuser"),
			),
			error: "Must be a team member",
		},
		{
			data: newTestData(map[string]string{
				"orgOwner": "orgOwner",
				"org":      "ReqTeamMembership",
				"teams":    "team2:otheruser",
				"team":     "team2",
			}, newSharedData().
				SetDoerName("regularuser"),
			),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"org": "ReqTeamMembership",
			}, newSharedData()),
			error: "reqTeamMembership: unprepared context",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"TokenRequiresScopes",
		"ReqTeamMembership",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetOwnDefault("org", "ReqTeamMembership")
		data.SetOwnDefault("team", org_model.OwnerTeamName)
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		orgOwner := data.shared.DoerName()
		if data.HasOwn("orgOwner") {
			orgOwner = data.GetOwn("orgOwner")
		}
		var org *org_model.Organization
		if data.HasOwn("org") {
			fixtureCreateUser(t, &user_model.User{Name: orgOwner})
			org = fixtureCreateOrg(t, &org_model.Organization{Name: data.GetOwn("org")}, &user_model.User{Name: orgOwner})
		}

		if data.HasOwn("teams") {
			fixtureCreateTeams(t, org, data.GetOwn("teams"))
		}

		if data.HasOwn("team") {
			team, err := org_model.GetTeam(t.Context(), org.ID, data.GetOwn("team"))
			require.NoError(t, err)
			permissions.SetTeam(team)
		}
	},
})
