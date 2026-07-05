// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	"forgejo.org/modules/setting"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqBasicOrRevProxyAuth, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"Service.EnableReverseProxyAuthAPI": "true",
			}, map[string]string{
				"doer":           "regularuser",
				"authentication": "proxy",
			}),
		},
		{
			data: newTestData(map[string]string{
				"Service.EnableReverseProxyAuthAPI": "false",
			}, map[string]string{
				"doer":           "regularuser",
				"authentication": "basic",
			}),
		},
		{
			data: newTestData(map[string]string{
				"Service.EnableReverseProxyAuthAPI": "true",
			}, map[string]string{
				"doer":           "regularuser",
				"authentication": "token",
			}),
			error: "auth method not allowed",
		},
		{
			data: newTestData(map[string]string{
				"Service.EnableReverseProxyAuthAPI": "false",
			}, map[string]string{
				"doer":           "regularuser",
				"authentication": "token",
			}),
			error: "auth method not allowed",
		},
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetSharedDefault("doer", "regularuser")
		data.SetSharedDefault("authentication", "proxy")
		data.SetOwnDefault("Service.EnableReverseProxyAuthAPI", "true")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		fixtureSetDoer(t, permissions, data)
		setting.Service.EnableReverseProxyAuthAPI = data.GetOwn("Service.EnableReverseProxyAuthAPI") == "true"
	},
})
