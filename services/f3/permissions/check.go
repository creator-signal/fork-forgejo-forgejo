// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	"context"

	f3_context "forgejo.org/services/f3/context"
	permissions_context "forgejo.org/services/permissions/context"
)

func Check(ctx context.Context, f permissions_context.CheckFunc) error {
	if permissionsCtx := f3_context.Get(ctx); permissionsCtx != nil {
		return f(permissionsCtx)
	}
	return nil
}
