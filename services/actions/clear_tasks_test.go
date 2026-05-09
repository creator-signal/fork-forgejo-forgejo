// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelAbandonedJobs(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestCancelAbandonedJobs")()
	require.NoError(t, unittest.PrepareTestDatabase())

	require.NoError(t, CancelAbandonedJobs(t.Context()))

	// status waiting, too long, ready to be abandoned
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 600})
	assert.Equal(t, actions_model.StatusCancelled, job.Status)

	// status blocked, too long, ready to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 601})
	assert.Equal(t, actions_model.StatusCancelled, job.Status)

	// status blocked, *not* too long, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 602})
	assert.Equal(t, actions_model.StatusBlocked, job.Status)

	// status running, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 603})
	assert.Equal(t, actions_model.StatusRunning, job.Status)

	// related run needs approval, not to be abandoned
	job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 604})
	assert.Equal(t, actions_model.StatusWaiting, job.Status)
}

// TestStopZombieTasks reproduces the false-positive zombie-detection
// bug: a long-running task whose `Updated` column is older than
// ZombieTaskTimeout but whose owning runner is still alive (recent
// LastOnline) must NOT be killed by the reaper. Conversely, a task
// whose runner has been silent for ages SHOULD be killed.
//
// Two paired tasks under one run cover both cases in a single pass:
//   - task 700 / runner 1001 — runner is bumped to LastOnline=now()
//     in this test, simulating a runner that's still pinging the
//     server (Ping/FetchTask) even though one specific task's
//     UpdateTask got stuck under load.
//   - task 701 / runner 1002 — runner stays at LastOnline=0 (the
//     fixture default), simulating a genuinely dead runner.
//
// On unfixed code the first assertion fails: the reaper marks task
// 700 as Failure even though the runner is online. After teaching
// StopZombieTasks to short-circuit on a recent runner heartbeat,
// both assertions pass.
func TestStopZombieTasks(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestStopZombieTasks")()
	require.NoError(t, unittest.PrepareTestDatabase())

	// Bump runner 1001 (the "alive" one) to right-now after the
	// fixture load so the YAML can stay deterministic. Runner 1002
	// stays at LastOnline=0 (1970) — a runner that has never spoken.
	runner := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunner{ID: 1001})
	runner.LastOnline = timeutil.TimeStampNow()
	_, err := db.GetEngine(t.Context()).ID(runner.ID).Cols("last_online").Update(runner)
	require.NoError(t, err)

	require.NoError(t, StopZombieTasks(t.Context()))

	// Task 700: runner is alive — must remain Running. The whole
	// point of this test is that this assertion is the bug
	// reproducer; it fails on the upstream code that ignores
	// runner.LastOnline when reaping zombies.
	task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 700})
	assert.Equal(t, actions_model.StatusRunning, task.Status,
		"task with a recently-online runner must NOT be killed by the zombie reaper")

	// Task 701: runner is offline (LastOnline=0) — sanity check
	// that legitimate zombies are still cleaned up.
	task = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 701})
	assert.Equal(t, actions_model.StatusFailure, task.Status,
		"task with an offline runner SHOULD be marked failure")
}
