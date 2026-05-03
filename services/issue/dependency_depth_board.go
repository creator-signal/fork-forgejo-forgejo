// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package issue

import (
	"context"
	"sort"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/convert"
)

type DepthBoardContext struct {
	Response   *api.DepthBoardResponse
	Issues     issues_model.IssueList
	Successors map[int64][]int64
}

type boardIssueGraph struct {
	IDs          []int64
	IssueData    map[int64]*api.DepthBoardIssue
	Dependencies map[int64][]int64
}

func BuildBoardColumns(
	result *DepthResult,
	issueData map[int64]*api.DepthBoardIssue,
	msCards []*api.DepthBoardMilestone,
) []*api.DepthBoardColumn {
	depthGroups := make(map[int][]int64)
	for id, d := range result.Depths {
		depthGroups[d] = append(depthGroups[d], id)
	}

	var columns []*api.DepthBoardColumn
	for d := 0; d <= result.MaxDepth; d++ {
		ids, ok := depthGroups[d]
		if !ok {
			continue
		}
		sort.SliceStable(ids, func(i, j int) bool {
			return result.InDegree[ids[i]] > result.InDegree[ids[j]]
		})

		var issues []*api.DepthBoardIssue
		for _, id := range ids {
			if issue, ok := issueData[id]; ok {
				issues = append(issues, issue)
			}
		}

		columns = append(columns, &api.DepthBoardColumn{
			Depth:  d,
			Issues: issues,
		})
	}

	if len(msCards) > 0 {
		columns = append(columns, &api.DepthBoardColumn{
			IsMilestone: true,
			Milestones:  msCards,
		})
	}

	return columns
}

func milestoneToCard(ms *issues_model.Milestone) *api.DepthBoardMilestone {
	return &api.DepthBoardMilestone{
		Milestone:    convert.ToAPIMilestone(ms),
		Completeness: ms.Completeness,
		IsOverdue:    ms.IsOverdue,
	}
}

func loadMilestoneCards(ctx context.Context, repoID int64) ([]*api.DepthBoardMilestone, error) {
	allRepoMilestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptionsAll,
		RepoID:      repoID,
	})
	if err != nil {
		return nil, err
	}

	var msCards []*api.DepthBoardMilestone
	for _, ms := range allRepoMilestones {
		msCards = append(msCards, milestoneToCard(ms))
	}
	sort.Slice(msCards, func(i, j int) bool {
		si, sj := msCards[i].State, msCards[j].State
		if si == api.StateOpen && sj == api.StateClosed {
			return true
		}
		if si == api.StateClosed && sj == api.StateOpen {
			return false
		}
		return msCards[i].ID < msCards[j].ID
	})
	return msCards, nil
}

func filterBlockedIssues(g *boardIssueGraph) {
	blockedSet := make(map[int64]bool, len(g.IDs))
	for id, deps := range g.Dependencies {
		if len(deps) > 0 {
			blockedSet[id] = true
		}
	}
	n := 0
	for _, id := range g.IDs {
		if !blockedSet[id] {
			g.IDs[n] = id
			n++
		}
	}
	g.IDs = g.IDs[:n]
	for id := range blockedSet {
		delete(g.IssueData, id)
	}
	filteredDeps := make(map[int64][]int64, len(g.IDs))
	for _, id := range g.IDs {
		filteredDeps[id] = g.Dependencies[id]
	}
	g.Dependencies = filteredDeps
}

func GetDependencyDepthBoardData(
	ctx context.Context,
	doer *user_model.User,
	repo *repo_model.Repository,
	milestoneID int64,
	isClosed optional.Option[bool],
	hideBlocked bool,
) (*DepthBoardContext, error) {
	opts := &issues_model.IssuesOptions{
		RepoIDs: []int64{repo.ID},
	}
	if milestoneID > 0 {
		opts.MilestoneIDs = []int64{milestoneID}
	}
	if has, val := isClosed.Get(); has {
		opts.IsClosed = optional.Some(val)
	}

	issues, err := issues_model.Issues(ctx, opts)
	if err != nil {
		return nil, err
	}

	msCards, err := loadMilestoneCards(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		columns := BuildBoardColumns(&DepthResult{}, nil, msCards)
		return &DepthBoardContext{
			Response: &api.DepthBoardResponse{Columns: columns},
		}, nil
	}

	g := &boardIssueGraph{
		IDs:       make([]int64, len(issues)),
		IssueData: make(map[int64]*api.DepthBoardIssue, len(issues)),
	}

	for i, issue := range issues {
		g.IDs[i] = issue.ID
		boardIssue := &api.DepthBoardIssue{
			Issue: convert.ToIssue(ctx, doer, issue),
		}
		if issue.IsPull && issue.PullRequest != nil {
			boardIssue.MergeStatus = ComputeMergeStatus(ctx, issue)
		}
		g.IssueData[issue.ID] = boardIssue
	}

	g.Dependencies, err = issues_model.LoadIssueDependencies(ctx, g.IDs)
	if err != nil {
		return nil, err
	}

	if hideBlocked {
		filterBlockedIssues(g)
	}

	result := ComputeDependencyDepth(g.IDs, g.Dependencies)

	for id, boardIssue := range g.IssueData {
		boardIssue.DependentsCount = result.InDegree[id]
		boardIssue.DependsOn = g.Dependencies[id]
		boardIssue.Blocks = result.Successors[id]
	}

	columns := BuildBoardColumns(result, g.IssueData, msCards)

	return &DepthBoardContext{
		Response: &api.DepthBoardResponse{
			Columns: columns,
			Cycles:  result.Cycles,
		},
		Issues:     issues,
		Successors: result.Successors,
	}, nil
}

func ComputeMergeStatus(ctx context.Context, issue *issues_model.Issue) string {
	pr := issue.PullRequest
	if pr == nil {
		return ""
	}
	if pr.HasMerged {
		return "merged"
	}
	if pr.Status == issues_model.PullRequestStatusConflict {
		return "conflict"
	}
	headCommitStatuses, _, err := git_model.GetLatestCommitStatus(ctx, pr.HeadRepoID, pr.HeadCommitID, db.ListOptionsAll)
	if err != nil || len(headCommitStatuses) == 0 {
		return ""
	}
	worst := git_model.CalcCommitStatus(headCommitStatuses)
	if worst == nil {
		return ""
	}
	switch worst.State {
	case api.CommitStatusSuccess:
		return "checks_passed"
	case api.CommitStatusPending:
		return "checks_pending"
	case api.CommitStatusError, api.CommitStatusFailure:
		return "checks_failed"
	default:
		return ""
	}
}
