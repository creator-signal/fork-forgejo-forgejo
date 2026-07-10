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
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("root").
				SetDoerAdmin(true).
				SetRepositoryName("userowner/repositoryprivate").
				SetRepositoryPrivate(true),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositoryprivate").
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepositoryName("userowner/repositoryprivate").
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(user_model.ActionsUserName).
				SetDoerActions(true).
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(user_model.ActionsUserName).
				SetDoerActions(true).
				SetDoerActionsRepoID(111111111111).
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(user_model.ActionsUserName).
				SetDoerActions(true).
				SetDoerActionsIsForkPullRequest(true).
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.shared.SetRepositoryNameDefault("userowner/repositorypublic")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureSetRepository(t, permissions, data.shared.RepositoryName(), data.shared.RepositoryInit())
	},
})
