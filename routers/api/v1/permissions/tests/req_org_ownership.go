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

var _ = registerFunctionTest(apiv1_permissions.ReqOrgOwnership, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":    "ReqOrgOwnershipOrg",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":   "doeradmin",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":    "ReqOrgOwnershipOrg",
				"doer":   "regularuser",
				"setOrg": "true",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":      "ReqOrgOwnershipOrg",
				"orgOwner": "ReqOrgOwnershipOrgOwner",
				"doer":     "regularuser",
				"setOrg":   "true",
			}),
			error: "Must be an organization owner",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":     "ReqOrgOwnershipOrg",
				"doer":    "regularuser",
				"setTeam": "true",
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":      "ReqOrgOwnershipOrg",
				"orgOwner": "ReqOrgOwnershipOrgOwner",
				"doer":     "regularuser",
				"setTeam":  "true",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"setOrg": "true",
			}),
			error: "reqOrgOwnership: unprepared context",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"TokenRequiresScopes",
		"ReqOrgOwnership",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("org", "ReqOrgOwnershipOrg")
		data.SetSharedDefault("setOrg", "true")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		orgOwner := data.GetShared("doer")
		if data.HasShared("orgOwner") {
			orgOwner = data.GetShared("orgOwner")
		}
		var org *org_model.Organization
		if data.HasShared("org") {
			fixtureCreateUser(t, &user_model.User{Name: orgOwner})
			org = fixtureCreateOrg(t, &org_model.Organization{Name: data.GetShared("org")}, &user_model.User{Name: orgOwner})
		}

		if data.GetShared("setOrg") == "true" {
			permissions.SetOrganization(org)
		}

		if data.GetShared("setTeam") == "true" {
			team, err := org_model.GetTeam(t.Context(), org.ID, org_model.OwnerTeamName)
			require.NoError(t, err)
			permissions.SetTeam(team)
		}
	},
})
