// Copyright 2024 The Gitea Authors. All rights reserved.
// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAllRerunJobs(t *testing.T) {
	job1 := &actions_model.ActionRunJob{JobID: "job1"}
	job2 := &actions_model.ActionRunJob{JobID: "job2", Needs: []string{"job1"}}
	job3 := &actions_model.ActionRunJob{JobID: "job3", Needs: []string{"job2"}}
	job4 := &actions_model.ActionRunJob{JobID: "job4", Needs: []string{"job2", "job3"}}

	jobs := []*actions_model.ActionRunJob{job1, job2, job3, job4}

	testCases := []struct {
		job       *actions_model.ActionRunJob
		rerunJobs []*actions_model.ActionRunJob
	}{
		{
			job1,
			[]*actions_model.ActionRunJob{job1, job2, job3, job4},
		},
		{
			job2,
			[]*actions_model.ActionRunJob{job2, job3, job4},
		},
		{
			job3,
			[]*actions_model.ActionRunJob{job3, job4},
		},
		{
			job4,
			[]*actions_model.ActionRunJob{job4},
		},
	}

	for _, tc := range testCases {
		rerunJobs := GetAllRerunJobs(tc.job, jobs)
		assert.ElementsMatch(t, tc.rerunJobs, rerunJobs)
	}
}

func TestActions_RerunJobs(t *testing.T) {
	t.Run("RerunJob completed job becomes Waiting", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20000})
		require.Equal(t, actions_model.StatusSuccess, job.Status)
		require.Equal(t, int64(1), job.Attempt)

		require.NoError(t, RerunJob(t.Context(), job, false))

		job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20000})
		assert.Equal(t, actions_model.StatusWaiting, job.Status)
		assert.Equal(t, int64(2), job.Attempt)
		assert.Zero(t, job.TaskID)
		assert.Zero(t, job.Started)
		assert.Zero(t, job.Stopped)
	})

	t.Run("RerunJob completed job becomes Blocked when shouldBlock", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20001})
		require.Equal(t, actions_model.StatusSuccess, job.Status)

		require.NoError(t, RerunJob(t.Context(), job, true))

		job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20001})
		assert.Equal(t, actions_model.StatusBlocked, job.Status)
		assert.Equal(t, int64(2), job.Attempt)
		assert.Zero(t, job.TaskID)
	})

	t.Run("RerunJob running job is no-op", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20003})
		require.Equal(t, actions_model.StatusRunning, job.Status)

		require.NoError(t, RerunJob(t.Context(), job, false))

		// Should remain unchanged
		job = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20003})
		assert.Equal(t, actions_model.StatusRunning, job.Status)
		assert.Equal(t, int64(1), job.Attempt)
	})

	t.Run("RerunRunJobs all jobs rerun and run timing reset", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 1000})
		require.Equal(t, actions_model.StatusSuccess, run.Status)
		require.NotZero(t, run.Started)
		require.NotZero(t, run.Stopped)

		rerunJobs, err := RerunRunJobs(t.Context(), run, 0)
		require.NoError(t, err)
		assert.Len(t, rerunJobs, 3)

		// Run timing should be reset
		run = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 1000})
		assert.Zero(t, run.Started)
		assert.Zero(t, run.Stopped)

		// job_1 (no needs) should be Waiting
		job1 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20000})
		assert.Equal(t, actions_model.StatusWaiting, job1.Status)
		assert.Equal(t, int64(2), job1.Attempt)

		// job_2 (needs job_1) should be Blocked
		job2 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20001})
		assert.Equal(t, actions_model.StatusBlocked, job2.Status)
		assert.Equal(t, int64(2), job2.Attempt)

		// job_3 (needs job_2) should be Blocked
		job3 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20002})
		assert.Equal(t, actions_model.StatusBlocked, job3.Status)
		assert.Equal(t, int64(2), job3.Attempt)
	})

	t.Run("RerunRunJobs single job reruns with dependents", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 1000})

		// Rerun job_2 (ID 20001) — should also rerun job_3 (depends on job_2)
		rerunJobs, err := RerunRunJobs(t.Context(), run, 20001)
		require.NoError(t, err)
		assert.Len(t, rerunJobs, 2) // job_2 + job_3

		// job_1 should remain unchanged (not a dependent)
		job1 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20000})
		assert.Equal(t, actions_model.StatusSuccess, job1.Status)
		assert.Equal(t, int64(1), job1.Attempt)

		// job_2 (the target) should be Waiting
		job2 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20001})
		assert.Equal(t, actions_model.StatusWaiting, job2.Status)
		assert.Equal(t, int64(2), job2.Attempt)

		// job_3 (dependent of job_2) should be Blocked
		job3 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20002})
		assert.Equal(t, actions_model.StatusBlocked, job3.Status)
		assert.Equal(t, int64(2), job3.Attempt)
	})

	t.Run("RerunRunJobs running run does not reset timing", func(t *testing.T) {
		defer unittest.OverrideFixtures("services/actions/TestActions_RerunJobs")()
		require.NoError(t, unittest.PrepareTestDatabase())

		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 1100})
		require.Equal(t, actions_model.StatusRunning, run.Status)
		originalStarted := run.Started

		rerunJobs, err := RerunRunJobs(t.Context(), run, 0)
		require.NoError(t, err)
		// The running job should not be rerun
		assert.Len(t, rerunJobs, 1)

		// Run timing should NOT be reset (run is not done)
		run = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 1100})
		assert.Equal(t, originalStarted, run.Started)

		// Running job should remain unchanged
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 20003})
		assert.Equal(t, actions_model.StatusRunning, job.Status)
		assert.Equal(t, int64(1), job.Attempt)
	})
}
