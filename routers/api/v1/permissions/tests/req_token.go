// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqToken, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoerName("regularuser"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoerName(user_model.ActionsUserName).
				SetDoerActions(true),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true),
			),
			error: "token is required",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.shared.SetDoerNameDefault("doerregular")
	},
})
