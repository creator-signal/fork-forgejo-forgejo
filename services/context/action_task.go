// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package context

import (
	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/perm"
)

type ActionTask struct {
	RepoID  int64
	OwnerID int64

	TokenScope auth_model.AccessTokenScope
}

func ActionTaskAssignment() func(ctx *Context) {
	return func(ctx *Context) {
		if has, taskID := ctx.Authentication.ActionsTaskID().Get(); has {
			task, err := actions_model.GetTaskByID(ctx, taskID)
			if err != nil {
				ctx.ServerError("GetTaskByID", err);
				return
			}
			ctx.ActionTask = &ActionTask{
				RepoID:     task.RepoID,
				OwnerID:    task.OwnerID,
				TokenScope: task.TokenScope,
			}
		}
	}
}

func (t *ActionTask) AllowsOwnerAccess(ownerID int64) bool {
	return t.OwnerID == ownerID
}

func (t *ActionTask) AllowsRepoAccess(repoID int64) bool {
	return t.RepoID == repoID
}

func (t *ActionTask) getAccessMode(write, read auth_model.AccessTokenScope) (perm.AccessMode, error) {
	if isWrite, err := t.TokenScope.HasScope(write); err != nil {
		return perm.AccessModeNone, err
	} else if isWrite {
		return perm.AccessModeWrite, nil
	}

	if isRead, err := t.TokenScope.HasScope(read); err != nil {
		return perm.AccessModeNone, err
	} else if isRead {
		return perm.AccessModeRead, nil
	}

	return perm.AccessModeNone, nil
}

func (t *ActionTask) GetPackageAccessMode() (perm.AccessMode, error) {
	return t.getAccessMode(auth_model.AccessTokenScopeWritePackage, auth_model.AccessTokenScopeReadPackage)
}

func (t *ActionTask) CanWritePackageToRepo(repoID int64) (bool, error) {
	if t.AllowsRepoAccess(repoID) {
		if isWrite, err := t.TokenScope.HasScope(auth_model.AccessTokenScopeWritePackage); err != nil {
			return false, err
		} else if isWrite {
			return true, nil
		}
	}
	return false, nil
}
