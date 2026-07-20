// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"context"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// RunnerStats holds aggregated runner and task statistics.
// It is scope-aware: stats are filtered by ownerID/repoID to prevent data leaking across scopes.
type RunnerStats struct {
	TotalRunners  int64
	OnlineRunners int64
	IdleRunners   int64
	BusyRunners   int64

	// Task stats for the given period (default 30 days)
	TotalTasks      int64
	SuccessTasks    int64
	FailureTasks    int64
	CancelledTasks  int64
	SkippedTasks    int64
	SuccessRate     float64 // 0-100
	AvgDurationSecs int64
}

// RunnerStatsOptions configures the scope for runner statistics queries.
type RunnerStatsOptions struct {
	// OwnerID filters stats to runners/tasks owned by this user/org. 0 = no filter.
	OwnerID int64
	// RepoID filters stats to runners/tasks belonging to this repository. 0 = no filter.
	RepoID int64
	// Since defines the start of the time window for task stats. Zero = 30 days ago.
	Since timeutil.TimeStamp
}

// GetRunnerStats computes runner utilization statistics using SQL aggregation.
// It respects scope boundaries (global/org/repo) so numbers don't leak across contexts.
func GetRunnerStats(ctx context.Context, opts RunnerStatsOptions) (*RunnerStats, error) {
	stats := &RunnerStats{}

	if opts.Since == 0 {
		opts.Since = timeutil.TimeStamp(time.Now().AddDate(0, 0, -30).Unix())
	}

	// Count runners by status using the runner list
	runners, err := db.Find[ActionRunner](ctx, FindRunnerOptions{
		OwnerID: opts.OwnerID,
		RepoID:  opts.RepoID,
	})
	if err != nil {
		return nil, err
	}

	stats.TotalRunners = int64(len(runners))
	for _, r := range runners {
		if r.IsOnline() {
			stats.OnlineRunners++
			if r.IsActive() {
				stats.BusyRunners++
			} else {
				stats.IdleRunners++
			}
		}
	}

	// Task stats via SQL aggregation - avoids loading all tasks into memory
	type taskStatusCount struct {
		Status Status `xorm:"status"`
		Count  int64  `xorm:"count"`
	}

	e := db.GetEngine(ctx)
	var counts []taskStatusCount

	sess := e.Table("action_task").
		Select("status, COUNT(*) AS count").
		Where("started >= ?", opts.Since)

	if opts.OwnerID > 0 {
		sess = sess.And("owner_id = ?", opts.OwnerID)
	}
	if opts.RepoID > 0 {
		sess = sess.And("repo_id = ?", opts.RepoID)
	}

	err = sess.GroupBy("status").Find(&counts)
	if err != nil {
		return nil, err
	}

	for _, c := range counts {
		stats.TotalTasks += c.Count
		switch c.Status {
		case StatusSuccess:
			stats.SuccessTasks = c.Count
		case StatusFailure:
			stats.FailureTasks = c.Count
		case StatusCancelled:
			stats.CancelledTasks = c.Count
		case StatusSkipped:
			stats.SkippedTasks = c.Count
		}
	}

	if stats.TotalTasks > 0 {
		stats.SuccessRate = float64(stats.SuccessTasks) * 100 / float64(stats.TotalTasks)
	}

	// Average duration via SQL - only for completed tasks (success + failure)
	type avgResult struct {
		AvgDuration float64 `xorm:"avg_duration"`
	}
	var avg avgResult

	avgSess := e.Table("action_task").
		Select("AVG(stopped - started) AS avg_duration").
		Where("started >= ?", opts.Since).
		And("stopped > started").
		And("status IN (?, ?)", StatusSuccess, StatusFailure)

	if opts.OwnerID > 0 {
		avgSess = avgSess.And("owner_id = ?", opts.OwnerID)
	}
	if opts.RepoID > 0 {
		avgSess = avgSess.And("repo_id = ?", opts.RepoID)
	}

	_, err = avgSess.Get(&avg)
	if err != nil {
		return nil, err
	}
	stats.AvgDurationSecs = int64(avg.AvgDuration)

	return stats, nil
}
