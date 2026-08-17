// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	"context"

	"forgejo.org/modules/setting"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
	apiv1_permissions_testhelpers "forgejo.org/routers/api/v1/permissions/testhelpers"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions_api "forgejo.org/services/f3/permissions/api"
)

type CheckFunc func(*f3_context.F3)

func Check(ctx context.Context, f CheckFunc, methodsAndPattern string) {
	f3Ctx := f3_context.Get(ctx)
	if f3Ctx == nil {
		panic("no F3 context")
	}
	if setting.IsInTesting {
		f3_permissions_api.ResetPermissionsCheckCalled()
		// APIAuthorization is called once when F3 is initialized. It is implicitly the first permission function to run in all cases
		// The Forgejo REST API is different and will explicitly run this permission function for all endpoint
		// It is added here to allow a strict comparison between the check of a Forgejo REST API endpoint and a F3 internal method
		f3_permissions_api.AddPermissionsCheckCall(apiv1_permissions.APIAuthorization)
	}
	f(f3Ctx)
	if setting.IsInTesting {
		apiv1_permissions_testhelpers.MustMatch(f, f3_permissions_api.GetPermissionsCheckCalled(), methodsAndPattern)
	}
}
