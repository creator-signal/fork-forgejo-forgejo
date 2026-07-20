// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package metrics

import (
	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"

	"github.com/prometheus/client_golang/prometheus"
)

// ActionsRunnerCollector implements prometheus.Collector for Actions runner metrics.
type ActionsRunnerCollector struct {
	RunnersTotal  *prometheus.Desc
	RunnersOnline *prometheus.Desc
	RunnersBusy   *prometheus.Desc

	TasksTotal    *prometheus.Desc
	TasksDuration *prometheus.Desc
}

// NewActionsRunnerCollector creates a new collector for runner utilization metrics.
func NewActionsRunnerCollector() ActionsRunnerCollector {
	return ActionsRunnerCollector{
		RunnersTotal: prometheus.NewDesc(
			namespace+"actions_runners_total",
			"Total number of registered Actions runners",
			nil, nil,
		),
		RunnersOnline: prometheus.NewDesc(
			namespace+"actions_runners_online",
			"Number of Actions runners currently online",
			nil, nil,
		),
		RunnersBusy: prometheus.NewDesc(
			namespace+"actions_runners_busy",
			"Number of Actions runners currently executing a task",
			nil, nil,
		),
		TasksTotal: prometheus.NewDesc(
			namespace+"actions_tasks_total",
			"Total number of Actions tasks in the last 30 days",
			[]string{"status"}, nil,
		),
		TasksDuration: prometheus.NewDesc(
			namespace+"actions_tasks_duration_seconds_avg",
			"Average duration of completed Actions tasks in the last 30 days",
			nil, nil,
		),
	}
}

// Describe sends the descriptors of each metric to the provided channel.
func (c ActionsRunnerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.RunnersTotal
	ch <- c.RunnersOnline
	ch <- c.RunnersBusy
	ch <- c.TasksTotal
	ch <- c.TasksDuration
}

// Collect fetches runner stats and delivers them as Prometheus metrics.
func (c ActionsRunnerCollector) Collect(ch chan<- prometheus.Metric) {
	stats, err := actions_model.GetRunnerStats(db.DefaultContext, actions_model.RunnerStatsOptions{})
	if err != nil {
		// Cannot collect metrics if the query fails
		return
	}

	ch <- prometheus.MustNewConstMetric(
		c.RunnersTotal,
		prometheus.GaugeValue,
		float64(stats.TotalRunners),
	)
	ch <- prometheus.MustNewConstMetric(
		c.RunnersOnline,
		prometheus.GaugeValue,
		float64(stats.OnlineRunners),
	)
	ch <- prometheus.MustNewConstMetric(
		c.RunnersBusy,
		prometheus.GaugeValue,
		float64(stats.BusyRunners),
	)
	ch <- prometheus.MustNewConstMetric(
		c.TasksTotal,
		prometheus.GaugeValue,
		float64(stats.SuccessTasks),
		"success",
	)
	ch <- prometheus.MustNewConstMetric(
		c.TasksTotal,
		prometheus.GaugeValue,
		float64(stats.FailureTasks),
		"failure",
	)
	ch <- prometheus.MustNewConstMetric(
		c.TasksTotal,
		prometheus.GaugeValue,
		float64(stats.CancelledTasks),
		"cancelled",
	)
	ch <- prometheus.MustNewConstMetric(
		c.TasksTotal,
		prometheus.GaugeValue,
		float64(stats.SkippedTasks),
		"skipped",
	)
	ch <- prometheus.MustNewConstMetric(
		c.TasksDuration,
		prometheus.GaugeValue,
		float64(stats.AvgDurationSecs),
	)
}
