// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanup(t *testing.T) {
	t.Run("Deletes no longer existing logs", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())

		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 1001, LogExpired: false, LogIndexes: []int64{1, 2, 3, 4}, LogFilename: "does-not-exist", Stopped: timeutil.TimeStamp(1)})

		require.NoError(t, CleanupLogs(db.DefaultContext))

		task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 1001})
		assert.Equal(t, "does-not-exist", task.LogFilename)
		assert.True(t, task.LogExpired)
		assert.Nil(t, task.LogIndexes)
	})

	t.Run("Expires tasks without logs", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())

		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 1001, LogExpired: false, LogIndexes: []int64{}, LogFilename: "", Stopped: timeutil.TimeStamp(1)})

		require.NoError(t, CleanupLogs(db.DefaultContext))

		task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 1001})
		assert.Empty(t, task.LogFilename)
		assert.True(t, task.LogExpired)
		assert.Nil(t, task.LogIndexes)
	})

	t.Run("Terminates on a full batch of tasks without logs", func(t *testing.T) {
		require.NoError(t, unittest.PrepareTestDatabase())

		// A task without logs used to be skipped without being marked as expired.
		// FindOldTasksToExpire selects on log_expired and has no ORDER BY, and the
		// loop only stops on a short batch, so a full batch of such tasks was
		// selected again on every iteration and the cleanup never returned.
		for i := range deleteLogBatchSize {
			unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{
				ID:          int64(1001 + i),
				LogExpired:  false,
				LogFilename: "",
				TokenHash:   fmt.Sprintf("cleanup-logs-token-%d", i),
				Stopped:     timeutil.TimeStamp(1),
			})
		}

		require.NoError(t, CleanupLogs(db.DefaultContext))

		for i := range deleteLogBatchSize {
			task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: int64(1001 + i)})
			assert.True(t, task.LogExpired)
		}
	})
}

func TestCleanupEphemeralRunners(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Deletes ephemeral runner with successful task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2001, UUID: "2001-uuid", TokenHash: "2001-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2001, RunnerID: 2001, TokenHash: "task-2001-hash", Status: actions_model.StatusSuccess})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertNotExistsBean(t, &actions_model.ActionRunner{ID: 2001})
	})

	t.Run("Deletes ephemeral runner with failed task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2002, UUID: "2002-uuid", TokenHash: "2002-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2002, RunnerID: 2002, TokenHash: "task-2002-hash", Status: actions_model.StatusFailure})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertNotExistsBean(t, &actions_model.ActionRunner{ID: 2002})
	})

	t.Run("Deletes ephemeral runner with cancelled task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2003, UUID: "2003-uuid", TokenHash: "2003-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2003, RunnerID: 2003, TokenHash: "task-2003-hash", Status: actions_model.StatusCancelled})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertNotExistsBean(t, &actions_model.ActionRunner{ID: 2003})
	})

	t.Run("Deletes ephemeral runner with skipped task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2004, UUID: "2004-uuid", TokenHash: "2004-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2004, RunnerID: 2004, TokenHash: "task-2004-hash", Status: actions_model.StatusSkipped})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertNotExistsBean(t, &actions_model.ActionRunner{ID: 2004})
	})

	t.Run("Keeps ephemeral runner with running task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2005, UUID: "2005-uuid", TokenHash: "2005-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2005, RunnerID: 2005, TokenHash: "task-2005-hash", Status: actions_model.StatusRunning})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunner{ID: 2005})
	})

	t.Run("Keeps ephemeral runner with waiting task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2006, UUID: "2006-uuid", TokenHash: "2006-hash", Ephemeral: true})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2006, RunnerID: 2006, TokenHash: "task-2006-hash", Status: actions_model.StatusWaiting})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunner{ID: 2006})
	})

	t.Run("Keeps ephemeral runner with no tasks", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2007, UUID: "2007-uuid", TokenHash: "2007-hash", Ephemeral: true})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunner{ID: 2007})
	})

	t.Run("Keeps non-ephemeral runner with completed task", func(t *testing.T) {
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionRunner{ID: 2008, UUID: "2008-uuid", TokenHash: "2008-hash", Ephemeral: false})
		unittest.AssertSuccessfulInsert(t, &actions_model.ActionTask{ID: 2008, RunnerID: 2008, TokenHash: "task-2008-hash", Status: actions_model.StatusSuccess})

		require.NoError(t, CleanupEphemeralRunners(db.DefaultContext))

		unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunner{ID: 2008})
	})
}
