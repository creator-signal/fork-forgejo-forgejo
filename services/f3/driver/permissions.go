// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	f3_permissions "forgejo.org/services/f3/permissions"
	permissions_context "forgejo.org/services/permissions/context"
)

func permissionsCheck(ctx context.Context, f permissions_context.CheckFunc) {
	if err := f3_permissions.Check(ctx, f); err != nil {
		panic(err)
	}
}
