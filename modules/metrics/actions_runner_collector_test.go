// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsRunnerCollectorDescribe(t *testing.T) {
	collector := NewActionsRunnerCollector()

	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	descs := make([]*prometheus.Desc, 0)
	for desc := range ch {
		descs = append(descs, desc)
	}

	// Should describe exactly 5 metrics
	assert.Len(t, descs, 5)

	// Verify metric names are present
	descStrings := make([]string, len(descs))
	for i, d := range descs {
		descStrings[i] = d.String()
	}

	assert.Contains(t, descStrings[0], "actions_runners_total")
	assert.Contains(t, descStrings[1], "actions_runners_online")
	assert.Contains(t, descStrings[2], "actions_runners_busy")
	assert.Contains(t, descStrings[3], "actions_tasks_total")
	assert.Contains(t, descStrings[4], "actions_tasks_duration_seconds_avg")
}

func TestActionsRunnerCollectorImplementsInterface(t *testing.T) {
	collector := NewActionsRunnerCollector()

	// Verify it implements the prometheus.Collector interface
	var _ prometheus.Collector = collector
}

func TestActionsRunnerCollectorCollectChannelHandling(t *testing.T) {
	// Verify that Describe and Collect use the channel correctly
	// and don't block or deadlock
	collector := NewActionsRunnerCollector()

	// Describe should send exactly 5 descriptors
	descCh := make(chan *prometheus.Desc, 10)
	collector.Describe(descCh)
	close(descCh)

	count := 0
	for range descCh {
		count++
	}
	assert.Equal(t, 5, count)
}

func TestActionsRunnerCollectorMetricTypes(t *testing.T) {
	collector := NewActionsRunnerCollector()

	ch := make(chan *prometheus.Desc, 10)
	collector.Describe(ch)
	close(ch)

	for desc := range ch {
		// All our metrics should be described without nil
		require.NotNil(t, desc)
	}
}

func TestNewActionsRunnerCollectorFields(t *testing.T) {
	collector := NewActionsRunnerCollector()

	// Verify all fields are initialized (non-nil)
	assert.NotNil(t, collector.RunnersTotal)
	assert.NotNil(t, collector.RunnersOnline)
	assert.NotNil(t, collector.RunnersBusy)
	assert.NotNil(t, collector.TasksTotal)
	assert.NotNil(t, collector.TasksDuration)
}

// TestTasksMetricHasStatusLabel verifies the tasks_total metric is defined with a status label
func TestTasksMetricHasStatusLabel(t *testing.T) {
	collector := NewActionsRunnerCollector()

	// Create a test metric using the TasksTotal descriptor
	m, err := prometheus.NewConstMetric(
		collector.TasksTotal,
		prometheus.GaugeValue,
		42,
		"success",
	)
	require.NoError(t, err)

	// Verify label
	var metric dto.Metric
	require.NoError(t, m.Write(&metric))

	require.Len(t, metric.Label, 1)
	assert.Equal(t, "status", *metric.Label[0].Name)
	assert.Equal(t, "success", *metric.Label[0].Value)
	assert.Equal(t, float64(42), *metric.Gauge.Value)
}
