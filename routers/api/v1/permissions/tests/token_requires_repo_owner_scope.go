// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"strings"
	"testing"

	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"

	"github.com/stretchr/testify/require"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.TokenRequiresRepoOwnerScope, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"owner": "doername",
			}, newSharedData().
				SetDoerScope("read:user").
				SetTokenLevel("read"),
			),
		},
		{
			data: newTestData(map[string]string{
				"owner": "doername",
			}, newSharedData().
				SetDoerScope("read:user").
				SetTokenLevel("write"),
			),
			error: "token does not have at least one of required scope(s): [write:user]",
		},
		{
			data: newTestData(map[string]string{
				"owner": "someorg",
			}, newSharedData().
				SetDoerScope("read:organization").
				SetTokenLevel("read"),
			),
		},
		{
			data: newTestData(map[string]string{
				"owner": "someorg",
			}, newSharedData().
				SetDoerScope("read:organization").
				SetTokenLevel("write"),
			),
			error: "token does not have at least one of required scope(s): [write:organization]",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		if !data.Has("owner") {
			if data.shared.HasRepositoryName() {
				owner, _, found := strings.Cut(data.shared.RepositoryName(), "/")
				require.True(t, found)
				data.Set("owner", owner)
			} else {
				data.Set("owner", "doername")
			}
		}
		data.shared.SetTokenLevelDefault("read")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		ownerName := data.Get("owner")
		if strings.Contains(ownerName, "org") {
			fixtureCreateOrg(t, &org_model.Organization{Name: ownerName}, &user_model.User{Name: "orgOwner" + ownerName})
			require.NotNil(t, fixtureGetUser(t, ownerName))
		} else {
			fixtureCreateUser(t, &user_model.User{Name: ownerName})
		}
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		owner := fixtureGetUser(t, data.Get("owner"))
		level := levelStringToLevel(data.shared.TokenLevel())
		t.Logf("calling TokenRequiresRepoOwnerScope(ctx, %+v, %v)", owner, level)
		apiv1_permissions.TokenRequiresRepoOwnerScope(ctx, owner, level)
	},
})
