// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"testing"

	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
)

// Issues are grouped into depth columns reflecting their distance from the
// root of the dependency graph; deeper dependencies appear in later columns.
func TestBuildBoardColumnsBasic(t *testing.T) {
	result := &DepthResult{
		Depths:   map[int64]int{1: 2, 2: 1, 3: 0, 4: 0},
		InDegree: map[int64]int{1: 0, 2: 1, 3: 1, 4: 0},
		MaxDepth: 2,
	}
	issueData := map[int64]*api.DepthBoardIssue{
		1: {Issue: &api.Issue{ID: 1, Index: 1, Title: "A"}, DependentsCount: 0},
		2: {Issue: &api.Issue{ID: 2, Index: 2, Title: "B"}, DependentsCount: 1},
		3: {Issue: &api.Issue{ID: 3, Index: 3, Title: "C"}, DependentsCount: 1},
		4: {Issue: &api.Issue{ID: 4, Index: 4, Title: "D"}, DependentsCount: 0},
	}

	columns := BuildBoardColumns(result, issueData, nil)
	assert.Len(t, columns, 3)
	assert.Equal(t, 0, columns[0].Depth)
	assert.Equal(t, int64(3), columns[0].Issues[0].ID)
	assert.Equal(t, int64(4), columns[0].Issues[1].ID)
	assert.Equal(t, 1, columns[1].Depth)
	assert.Equal(t, int64(2), columns[1].Issues[0].ID)
	assert.Equal(t, 2, columns[2].Depth)
	assert.Equal(t, int64(1), columns[2].Issues[0].ID)
}

// When milestones are supplied, a dedicated milestone column is appended as
// the last column containing all milestone cards.
func TestBuildBoardColumnsWithMilestoneColumn(t *testing.T) {
	result := &DepthResult{
		Depths:   map[int64]int{1: 0, 2: 0},
		InDegree: map[int64]int{1: 0, 2: 0},
		MaxDepth: 0,
	}
	issueData := map[int64]*api.DepthBoardIssue{
		1: {Issue: &api.Issue{ID: 1, Index: 1, Title: "A", Milestone: &api.Milestone{ID: 10, Title: "v1.0"}}, DependentsCount: 0},
		2: {Issue: &api.Issue{ID: 2, Index: 2, Title: "B"}, DependentsCount: 0},
	}
	msCards := []*api.DepthBoardMilestone{
		{Milestone: &api.Milestone{ID: 10, Title: "v1.0", State: api.StateOpen}, Completeness: 50},
		{Milestone: &api.Milestone{ID: 20, Title: "v2.0", State: api.StateOpen}, Completeness: 0},
	}

	columns := BuildBoardColumns(result, issueData, msCards)
	assert.Len(t, columns, 2)
	lastCol := columns[len(columns)-1]
	assert.True(t, lastCol.IsMilestone)
	assert.Len(t, lastCol.Milestones, 2)
	assert.Equal(t, "v1.0", lastCol.Milestones[0].Title)
	assert.Equal(t, "v2.0", lastCol.Milestones[1].Title)
}

// An empty graph with no issues and no milestones produces no columns.
func TestBuildBoardColumnsEmpty(t *testing.T) {
	result := &DepthResult{
		Depths:   map[int64]int{},
		InDegree: map[int64]int{},
	}

	columns := BuildBoardColumns(result, nil, nil)
	assert.Empty(t, columns)
}

// When only milestones exist (no issues), the board produces a single
// milestone column.
func TestBuildBoardColumnsMilestonesOnlyNoIssues(t *testing.T) {
	msCards := []*api.DepthBoardMilestone{
		{Milestone: &api.Milestone{ID: 1, Title: "Alpha", State: api.StateOpen}},
		{Milestone: &api.Milestone{ID: 2, Title: "Beta", State: api.StateClosed}},
	}

	columns := BuildBoardColumns(&DepthResult{}, nil, msCards)
	assert.Len(t, columns, 1)
	assert.True(t, columns[0].IsMilestone)
	assert.Len(t, columns[0].Milestones, 2)
	assert.Equal(t, "Alpha", columns[0].Milestones[0].Title)
	assert.Equal(t, "Beta", columns[0].Milestones[1].Title)
}

// Within the same depth column, issues are sorted by dependent count in
// descending order so that the most-depended-on issues appear first.
func TestBuildBoardColumnsSortingByDependents(t *testing.T) {
	result := &DepthResult{
		Depths:   map[int64]int{1: 0, 2: 0, 3: 0},
		InDegree: map[int64]int{1: 5, 2: 3, 3: 10},
		MaxDepth: 0,
	}
	issueData := map[int64]*api.DepthBoardIssue{
		1: {Issue: &api.Issue{ID: 1, Index: 1, Title: "A"}, DependentsCount: 5},
		2: {Issue: &api.Issue{ID: 2, Index: 2, Title: "B"}, DependentsCount: 3},
		3: {Issue: &api.Issue{ID: 3, Index: 3, Title: "C"}, DependentsCount: 10},
	}

	columns := BuildBoardColumns(result, issueData, nil)
	assert.Len(t, columns, 1)
	assert.Equal(t, int64(3), columns[0].Issues[0].ID)
	assert.Equal(t, int64(1), columns[0].Issues[1].ID)
	assert.Equal(t, int64(2), columns[0].Issues[2].ID)
}

// Issues that depend on IDs not present in the graph (external/unresolved
// dependencies) are removed from the board.
func TestFilterBlockedIssuesRemovesBlocked(t *testing.T) {
	g := &boardIssueGraph{
		IDs: []int64{1, 2, 3, 4},
		IssueData: map[int64]*api.DepthBoardIssue{
			1: {Issue: &api.Issue{ID: 1, Title: "A"}},
			2: {Issue: &api.Issue{ID: 2, Title: "B"}},
			3: {Issue: &api.Issue{ID: 3, Title: "C"}},
			4: {Issue: &api.Issue{ID: 4, Title: "D"}},
		},
		Dependencies: map[int64][]int64{
			1: {10},
			2: {},
			3: {20},
			4: {},
		},
	}

	filterBlockedIssues(g)
	assert.ElementsMatch(t, []int64{2, 4}, g.IDs)
	assert.Contains(t, g.IssueData, int64(2))
	assert.Contains(t, g.IssueData, int64(4))
	assert.NotContains(t, g.IssueData, int64(1))
	assert.NotContains(t, g.IssueData, int64(3))
	for _, id := range g.IDs {
		assert.Contains(t, g.Dependencies, id)
	}
}

// When no issues have external dependencies, all issues are retained.
func TestFilterBlockedIssuesNoneBlocked(t *testing.T) {
	g := &boardIssueGraph{
		IDs: []int64{1, 2},
		IssueData: map[int64]*api.DepthBoardIssue{
			1: {Issue: &api.Issue{ID: 1, Title: "A"}},
			2: {Issue: &api.Issue{ID: 2, Title: "B"}},
		},
		Dependencies: map[int64][]int64{
			1: {},
			2: {},
		},
	}

	filterBlockedIssues(g)
	assert.ElementsMatch(t, []int64{1, 2}, g.IDs)
	assert.Len(t, g.IssueData, 2)
	assert.Len(t, g.Dependencies, 2)
}

// When every issue has an external dependency, the graph is emptied
// completely.
func TestFilterBlockedIssuesAllBlocked(t *testing.T) {
	g := &boardIssueGraph{
		IDs: []int64{1, 2},
		IssueData: map[int64]*api.DepthBoardIssue{
			1: {Issue: &api.Issue{ID: 1, Title: "A"}},
			2: {Issue: &api.Issue{ID: 2, Title: "B"}},
		},
		Dependencies: map[int64][]int64{
			1: {10},
			2: {20},
		},
	}

	filterBlockedIssues(g)
	assert.Empty(t, g.IDs)
	assert.Empty(t, g.IssueData)
	assert.Empty(t, g.Dependencies)
}

// Filtering an empty graph is a no-op.
func TestFilterBlockedIssuesEmpty(t *testing.T) {
	g := &boardIssueGraph{}
	filterBlockedIssues(g)
	assert.Empty(t, g.IDs)
	assert.Empty(t, g.IssueData)
	assert.Empty(t, g.Dependencies)
}
