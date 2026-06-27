// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package permissions

import (
	"net/http"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	"forgejo.org/models/unit"
	user_model "forgejo.org/models/user"
)

func RepoAccess(ctx Context) {
	if ctx.Doer() != nil && ctx.Doer().ID == user_model.ActionsUserID && ctx.Authentication().ActionsTaskID().Has() {
		_, taskID := ctx.Authentication().ActionsTaskID().Get()
		task, err := actions_model.GetTaskByID(ctx.GetContext(), taskID)
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "actions_model.GetTaskByID", err)
			return
		}
		if task.RepoID != ctx.GetRepository().ID {
			ctx.NotFound()
			return
		}

		if task.IsForkPullRequest {
			ctx.GetPermission().AccessMode = perm.AccessModeRead
		} else {
			ctx.GetPermission().AccessMode = perm.AccessModeWrite
		}

		if err := ctx.GetRepository().LoadUnits(ctx.GetContext()); err != nil {
			ctx.Error(http.StatusInternalServerError, "LoadUnits", err)
			return
		}
		ctx.GetPermission().Units = ctx.GetRepository().Units
		ctx.GetPermission().UnitsMode = make(map[unit.Type]perm.AccessMode)
		for _, u := range ctx.GetRepository().Units {
			ctx.GetPermission().UnitsMode[u.Type] = ctx.GetPermission().AccessMode
		}
	} else {
		permission, err := access_model.GetUserRepoPermissionWithReducer(ctx.GetContext(), ctx.GetRepository(), ctx.Doer(), ctx.GetReducer())
		if err != nil {
			ctx.Error(http.StatusInternalServerError, "GetUserRepoPermissionWithReducer", err)
			return
		}
		ctx.SetPermission(&permission)
	}

	if !ctx.GetPermission().HasAccess() {
		ctx.NotFound()
		return
	}
}
