// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package user

import (
	user_model "forgejo.org/models/user"
	permissions_context "forgejo.org/services/permissions/context"
	permissions_helpers "forgejo.org/services/permissions/helpers"
)

func Put(user *user_model.User) permissions_context.CheckFunc {
	return permissions_helpers.IsDoerSiteAdmin
}
