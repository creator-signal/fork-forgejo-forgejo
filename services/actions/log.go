// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/actions"
)

func TransferLogs(ctx context.Context, task *actions_model.ActionTask) error {
	if task.LogInStorage {
		return nil
	}
	if exists, err := actions.ExistsLogs(ctx, task.LogFilename); !exists || err != nil {
		return err
	}
	remove, err := actions.TransferLogs(ctx, task.LogFilename)
	if err != nil {
		return err
	}
	task.LogInStorage = true
	if err := actions_model.UpdateTask(ctx, task, "log_in_storage"); err != nil {
		return err
	}
	remove()

	return nil
}
