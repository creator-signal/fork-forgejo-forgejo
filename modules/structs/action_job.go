// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

import (
	"time"
)

// ActionRunJobStep represents a step in an action run job
// swagger:model
type ActionRunJobStep struct {
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Started time.Time `json:"started"`
	Stopped time.Time `json:"stopped"`
}

// ActionRunJobResponse represents a detailed action run job with steps
// swagger:model
type ActionRunJobResponse struct {
	ID      int64     `json:"id"`
	RunID   int64     `json:"run_id"`
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Started time.Time `json:"started,omitempty"`
	Stopped time.Time `json:"stopped,omitempty"`
	JobID   string    `json:"job_id"`
	Needs   []string  `json:"needs"`
	TaskID  int64     `json:"task_id"`
	// Steps are only included if the job has started
	Steps []*ActionRunJobStep `json:"steps,omitempty"`
	// Run information
	Run *ActionRunSummary `json:"run"`
	// Total number of jobs in the run
	TotalJobs int `json:"total_jobs"`
}

// ActionRunSummary represents a summary of an action run
// swagger:model
type ActionRunSummary struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Status string `json:"status"`
}
