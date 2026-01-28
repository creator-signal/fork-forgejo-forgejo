// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package helpers

import (
	"fmt"

	permissions_context "forgejo.org/services/permissions/context"
)

func IsDoerSiteAdmin(permissionsCtx permissions_context.PermissionsContext) error {
	if !permissionsCtx.GetDoer().IsAdmin {
		return fmt.Errorf("%+v is not an instance admin", permissionsCtx.GetDoer())
	}
	return nil
}
