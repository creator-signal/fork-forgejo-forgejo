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

var _ = registerFunctionTest(apiv1_permissions.ReqOrgMembership, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"org":    "ReqOrgMembershipOrg",
				"setOrg": "true",
			}, map[string]string{}),
		},
		{
			data: newTestData(map[string]string{
				"setOrg": "true",
			}, map[string]string{
				"doer": "doeradmin",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":    "ReqOrgMembershipOrg",
				"setOrg": "true",
			}, map[string]string{
				"doer": "regularuser",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"setOrg":   "true",
			}, map[string]string{
				"doer": "regularuser",
			}),
			error: "Must be an organization member",
		},
		{
			data: newTestData(map[string]string{
				"org":     "ReqOrgMembershipOrg",
				"setTeam": "true",
			}, map[string]string{
				"doer": "regularuser",
			}),
		},
		{
			data: newTestData(map[string]string{
				"org":      "ReqOrgMembershipOrg",
				"orgOwner": "ReqOrgMembershipOrgOwner",
				"setTeam":  "true",
			}, map[string]string{
				"doer": "regularuser",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"setOrg": "true",
			}, map[string]string{}),
			error: "unprepared context",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"TokenRequiresScopes",
		"ReqOrgMembership",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetOwnDefault("org", "ReqOrgMembership")
		data.SetOwnDefault("setOrg", "true")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		orgOwner := data.GetShared("doer")
		if data.HasOwn("orgOwner") {
			orgOwner = data.GetOwn("orgOwner")
		}
		var org *org_model.Organization
		if data.HasOwn("org") {
			fixtureCreateUser(t, &user_model.User{Name: orgOwner})
			org = fixtureCreateOrg(t, &org_model.Organization{Name: data.GetOwn("org")}, &user_model.User{Name: orgOwner})
		}

		if data.GetOwn("setOrg") == "true" {
			permissions.SetOrganization(org)
		}

		if data.GetOwn("setTeam") == "true" {
			team, err := org_model.GetTeam(t.Context(), org.ID, org_model.OwnerTeamName)
			require.NoError(t, err)
			permissions.SetTeam(team)
		}
	},
})
