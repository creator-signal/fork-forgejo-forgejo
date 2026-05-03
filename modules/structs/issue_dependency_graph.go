// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package structs

// DepthBoardResponse is the JSON response for the dependency depth board API.
// It contains columns arranged by dependency depth, plus any detected dependency cycles.
type DepthBoardResponse struct {
	Columns []*DepthBoardColumn `json:"columns"`
	Cycles  [][]int64           `json:"cycles,omitempty"`
}

// DepthBoardColumn represents one column in the dependency depth board.
// Non-milestone columns correspond to a depth level (root issues at the highest depth).
// Milestone columns aggregate milestones across all issues on the board.
type DepthBoardColumn struct {
	Depth       int                    `json:"depth,omitempty"`
	IsMilestone bool                   `json:"is_milestone,omitempty"`
	Issues      []*DepthBoardIssue     `json:"issues,omitempty"`
	Milestones  []*DepthBoardMilestone `json:"milestones,omitempty"`
}

// DepthBoardIssue extends the base Issue with dependency board metadata:
// how many issues depend on it and the IDs it depends on or blocks.
type DepthBoardIssue struct {
	*Issue
	DependentsCount int     `json:"dependents_count"`
	DependsOn       []int64 `json:"depends_on,omitempty"`
	Blocks          []int64 `json:"blocks,omitempty"`
	MergeStatus     string  `json:"merge_status,omitempty"`
}

// DepthBoardMilestone extends the base Milestone with completeness percentage and overdue status
// for display in the milestone column of the dependency board.
type DepthBoardMilestone struct {
	*Milestone
	Completeness int  `json:"completeness"`
	IsOverdue    bool `json:"is_overdue,omitempty"`
}
