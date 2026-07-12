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
			// The authenticated doer 'doername' can access to the publicly
			// readable repository 'userowner/repositorypublic'
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			// An anonymous visitor can access to the publicly
			// readable repository 'userowner/repositorypublic'
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			// The admin user 'root' can access the private repository
			// 'userowner/repositoryname'
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("root").
				SetDoerAdmin(true).
				SetRepositoryName("userowner/repositoryname").
				SetRepositoryPrivate(true),
			),
		},
		{
			// The unprivileged authenticated user 'doername' is denied
			// access to the private repository 'userowner/repositoryname'
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositoryname").
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			// An anonymous visitor is denied
			// access to the private repository 'userowner/repositoryname'
			data: newTestData(map[string]string{}, newSharedData().
				SetAnonymous(true).
				SetRepositoryName("userowner/repositoryname").
				SetRepositoryPrivate(true),
			),
			error: "Not Found",
		},
		{
			// The Forgejo Actions user token can access the repository
			// 'userowner/repositorypublic' because it is bound to it
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(user_model.ActionsUserName).
				SetDoerActions(true).
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			// The Forgejo Actions user token cannot access the repository
			// 'userowner/repositorypublic' although it is publicly readable
			// because it is bound to a different repository
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer(user_model.ActionsUserName).
				SetDoerActions(true).
				SetDoerActionsRepoID(111111111111).
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "Not Found",
		},
		{
			// The Forgejo Actions user token can access the repository
			// 'userowner/repositorypublic' because it is bound to it
			// even when it was created from a forked pull request event
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
		fixtureSetRepository(t, permissions, data.shared.RepositoryName(), data.shared.RepositoryInit(), data.shared.RepositoryPrivate(), data.shared.RepositoryArchived())
	},
})
