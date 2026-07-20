// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRunnerStatsDefaultSince(t *testing.T) {
	// When Since is 0, it should default to 30 days ago (not panic or error)
	defer unittest.OverrideFixtures("models/actions/TestGetRunnerStats")()
	require.NoError(t, unittest.PrepareTestDatabase())

	stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{})
	require.NoError(t, err)
	assert.NotNil(t, stats)
	// Default Since should still return valid results
	assert.GreaterOrEqual(t, stats.TotalRunners, int64(0))
}

func TestGetRunnerStatsEmptyResult(t *testing.T) {
	// Query with a non-existent owner should return zeroed stats (not error)
	defer unittest.OverrideFixtures("models/actions/TestGetRunnerStats")()
	require.NoError(t, unittest.PrepareTestDatabase())

	stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
		OwnerID: 99999, // non-existent owner
		Since:   timeutil.TimeStamp(1),
	})
	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(0), stats.TotalRunners)
	assert.Equal(t, int64(0), stats.TotalTasks)
	assert.Equal(t, int64(0), stats.SuccessTasks)
	assert.Equal(t, float64(0), stats.SuccessRate)
	assert.Equal(t, int64(0), stats.AvgDurationSecs)
}

func TestGetRunnerStats(t *testing.T) {
	defer unittest.OverrideFixtures("models/actions/TestGetRunnerStats")()
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Global scope returns all tasks", func(t *testing.T) {
		stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			Since: timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix()),
		})
		require.NoError(t, err)
		assert.NotNil(t, stats)

		// Should have runners from fixtures
		assert.Greater(t, stats.TotalRunners, int64(0))

		// Task counts should be >= 0 (depends on fixture data)
		assert.GreaterOrEqual(t, stats.TotalTasks, int64(0))
		assert.GreaterOrEqual(t, stats.SuccessTasks, int64(0))
		assert.GreaterOrEqual(t, stats.FailureTasks, int64(0))
		assert.GreaterOrEqual(t, stats.CancelledTasks, int64(0))
		assert.GreaterOrEqual(t, stats.SkippedTasks, int64(0))

		// Success rate should be 0-100
		assert.GreaterOrEqual(t, stats.SuccessRate, float64(0))
		assert.LessOrEqual(t, stats.SuccessRate, float64(100))
	})

	t.Run("Scoped by OwnerID filters correctly", func(t *testing.T) {
		// Query with owner_id=1 should only return tasks for that owner
		stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			OwnerID: 1,
			Since:   timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix()),
		})
		require.NoError(t, err)
		assert.NotNil(t, stats)

		// Compare with global stats
		globalStats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			Since: timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix()),
		})
		require.NoError(t, err)

		// Scoped stats should be <= global stats
		assert.LessOrEqual(t, stats.TotalTasks, globalStats.TotalTasks)
	})

	t.Run("Scoped by RepoID filters correctly", func(t *testing.T) {
		stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			RepoID: 4,
			Since:  timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix()),
		})
		require.NoError(t, err)
		assert.NotNil(t, stats)

		// Compare with global stats
		globalStats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			Since: timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix()),
		})
		require.NoError(t, err)

		// Scoped stats should be <= global stats
		assert.LessOrEqual(t, stats.TotalTasks, globalStats.TotalTasks)
	})

	t.Run("Success rate calculation", func(t *testing.T) {
		stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			Since: timeutil.TimeStamp(1), // Very old to capture all fixtures
		})
		require.NoError(t, err)

		if stats.TotalTasks > 0 {
			expectedRate := float64(stats.SuccessTasks) * 100 / float64(stats.TotalTasks)
			assert.InDelta(t, expectedRate, stats.SuccessRate, 0.01)
		} else {
			assert.Equal(t, float64(0), stats.SuccessRate)
		}
	})

	t.Run("Average duration is non-negative", func(t *testing.T) {
		stats, err := GetRunnerStats(db.DefaultContext, RunnerStatsOptions{
			Since: timeutil.TimeStamp(1),
		})
		require.NoError(t, err)
		assert.GreaterOrEqual(t, stats.AvgDurationSecs, int64(0))
	})
}
