// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	unit_model "forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.MustEnableLocalIssuesIfIsIssue, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"issue":       "issue5000",
				"issueAuthor": "issueAuthor",
			}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{
				"issue":       "issue5000",
				"issueAuthor": "issueAuthor",
			}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic").
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues}),
			),
			error: "Not Found",
		},
		{ // does not fail because it is an issue instead of a pull request
			data: newTestData(map[string]string{
				"pullRequestAuthor": "userowner",
				"pullRequestBranch": "MustEnableLocalIssuesIfIsIssue",
				"pullRequest":       "MustEnableLocalIssuesIfIsIssue",
				"issue":             "MustEnableLocalIssuesIfIsIssue",
			}, newSharedData().
				SetDoerName("doerregular").
				SetRepositoryName("userowner/repositorypublic").
				SetRepositoryDisabledUnits([]unit_model.Type{unit_model.TypeIssues}).
				SetRepositoryInit(true),
			),
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetOwnDefault("issue", "issueOne")
		data.SetOwnDefault("issueAuthor", "issueAuthor")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureDisableUnits(t, permissions, data.shared.RepositoryDisabledUnits())
		if data.HasOwn("pullRequest") {
			require.True(t, data.HasOwn("pullRequestBranch"))
			fixtureCreateBranch(t, permissions, data.GetOwn("pullRequestBranch"))
			require.True(t, data.HasOwn("pullRequestAuthor"))
			require.True(t, data.HasOwn("pullRequest"))
			fixtureCreatePullRequest(t, permissions, data.GetOwn("pullRequest"), data.GetOwn("pullRequestAuthor"), data.GetOwn("pullRequestBranch"))
			require.Equal(t, data.GetOwn("issue"), data.GetOwn("pullRequest"))
		} else {
			issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.GetOwn("issueAuthor")})
			fixtureSetIssue(t, permissions, data.GetOwn("issue"), issueAuthor.Name)
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		index := fixtureGetIssue(t, data.GetOwn("issue")).Index
		t.Logf("calling MustEnableLocalIssuesIfIsIssue(ctx, %d)", index)
		apiv1_permissions.MustEnableLocalIssuesIfIsIssue(ctx, index)
	},
})
