// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	unit_model "forgejo.org/models/unit"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.MustEnableIssuesOrPulls, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic").
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues, unit_model.TypePullRequests}).
				SetRepositoryInit(true),
			),
			error: "Not Found",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.shared.SetRepositoryInit(true)
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
	},
})
