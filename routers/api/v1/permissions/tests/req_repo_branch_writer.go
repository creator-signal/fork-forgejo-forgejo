// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"strings"
	"testing"

	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.ReqRepoBranchWriter, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"pullRequestAuthor": "userowner",
				"pullRequestBranch": "ReqRepoBranchWriter",
				"pullRequest":       "ReqRepoBranchWriter",
			}, newSharedData().
				SetDoerName("userowner").
				SetRepositoryName("userowner/repositorypublic").
				SetRepositoryInit(true),
			),
		},
		{
			data: newTestData(map[string]string{
				"pullRequestAuthor": "userowner",
				"pullRequestBranch": "ReqRepoBranchWriter",
				"pullRequest":       "ReqRepoBranchWriter",
			}, newSharedData().
				SetDoerName("regularuser").
				SetRepositoryName("userowner/repositorypublic").
				SetRepositoryInit(true),
			),
			error: "user should have a permission to write to this branch",
		},
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		require.True(t, data.HasOwn("pullRequestBranch"))
		fixtureCreateBranch(t, permissions, data.GetOwn("pullRequestBranch"))
		require.True(t, data.HasOwn("pullRequestAuthor"))
		require.True(t, data.HasOwn("pullRequest"))
		fixtureCreatePullRequest(t, permissions, data.GetOwn("pullRequest"), data.GetOwn("pullRequestAuthor"), data.GetOwn("pullRequestBranch"))
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		owner, _, found := strings.Cut(data.shared.RepositoryName(), "/")
		require.True(t, found)
		data.shared.SetDoerName(owner)
		data.shared.SetRepositoryInitDefault(true)
		data.SetOwnDefault("pullRequestAuthor", owner)
		data.SetOwnDefault("pullRequestBranch", "ReqRepoBranchWriter")
		data.SetOwnDefault("pullRequest", "ReqRepoBranchWriter")
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		branch := data.GetOwn("pullRequestBranch")
		t.Logf("calling ReqRepoBranchWriter(ctx, %s)", branch)
		apiv1_permissions.ReqRepoBranchWriter(ctx, branch)
	},
})
