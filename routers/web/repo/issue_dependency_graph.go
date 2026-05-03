// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"fmt"
	"net/http"
	"strconv"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	project_model "forgejo.org/models/project"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/services/context"
	issue_service "forgejo.org/services/issue"
)

func DependencyBoard(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.issues.dependency_board.title")
	ctx.Data["PageIsDependencyBoard"] = true
	ctx.Data["PageIsIssueList"] = true
	ctx.Data["IsDependenciesEnabled"] = ctx.Repo.Repository.IsDependenciesEnabled(ctx)

	milestones, err := db.Find[issues_model.Milestone](ctx, issues_model.FindMilestoneOptions{
		ListOptions: db.ListOptionsAll,
		RepoID:      ctx.Repo.Repository.ID,
	})
	if err != nil {
		ctx.ServerError("FindMilestones", err)
		return
	}
	ctx.Data["Milestones"] = milestones

	ctx.HTML(http.StatusOK, "repo/issues/dependency_board")
}

func DependencyBoardData(ctx *context.Context) {
	if !ctx.Repo.Repository.IsDependenciesEnabled(ctx) {
		ctx.JSON(http.StatusOK, struct {
			Columns []any `json:"columns"`
		}{})
		return
	}

	// Parse query parameters used to filter the dependency board:
	// - "milestone": optional milestone ID, limits issues to those in the given milestone
	// - "state": optional issue state filter ("open" or "closed")
	var milestoneID int64
	if v := ctx.FormString("milestone"); v != "" {
		var err error
		milestoneID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid milestone ID"})
			return
		}
	}

	var isClosed optional.Option[bool]
	if v := ctx.FormString("state"); v == "open" {
		isClosed = optional.Some(false)
	} else if v == "closed" {
		isClosed = optional.Some(true)
	}

	hideBlocked := ctx.FormString("hide_blocked") == "1"

	boardCtx, err := issue_service.GetDependencyDepthBoardData(ctx, ctx.Doer, ctx.Repo.Repository, milestoneID, isClosed, hideBlocked)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, boardCtx.Response)
}

func DependencyBoardCard(ctx *context.Context) {
	if !ctx.Repo.Repository.IsDependenciesEnabled(ctx) {
		ctx.JSON(http.StatusNotFound, map[string]string{"error": "dependencies not enabled"})
		return
	}

	issueIDStr := ctx.Params(":id")
	issueID, err := strconv.ParseInt(issueIDStr, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"error": "invalid issue ID"})
		return
	}

	issue, err := issues_model.GetIssueByID(ctx, issueID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("issue %d not found", issueID)})
		return
	}

	if err := issue.LoadRepo(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := issue.LoadPoster(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := issue.LoadMilestone(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := issue.LoadLabels(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := issue.LoadAssignees(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := issue.LoadComments(ctx); err != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if issue.IsPull {
		if err := issue.LoadPullRequest(ctx); err != nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	linkedPRsMap := buildLinkedPRsMap(ctx, issues_model.IssueList{issue})

	mergeStatus := ""
	if issue.IsPull && issue.PullRequest != nil {
		mergeStatus = issue_service.ComputeMergeStatus(ctx.Req.Context(), issue)
	}

	tmplData := map[string]any{
		"Issue":       issue,
		"MergeStatus": mergeStatus,
		"Page": map[string]any{
			"Project":     &project_model.Project{CardType: project_model.CardTypeTextOnly},
			"Repository":  ctx.Repo.Repository,
			"LinkedPRs":   linkedPRsMap,
			"IsRepoAdmin": ctx.Repo.IsAdmin(),
		},
		"isPinnedIssueCard": false,
	}

	cardHTML, renderErr := ctx.RenderToHTML("repo/issue/card", tmplData)
	if renderErr != nil {
		ctx.JSON(http.StatusInternalServerError, map[string]string{"error": renderErr.Error()})
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{"card_html": string(cardHTML)})
}

func isLinkedPullReference(comment *issues_model.Comment) bool {
	return comment.RefIssueID != 0 && comment.RefIsPull
}

func buildLinkedPRsMap(ctx *context.Context, issues issues_model.IssueList) map[int64][]*issues_model.Issue {
	linkedPRsMap := make(map[int64][]*issues_model.Issue)
	for _, issue := range issues {
		var referencedIDs []int64
		for _, comment := range issue.Comments {
			if isLinkedPullReference(comment) {
				referencedIDs = append(referencedIDs, comment.RefIssueID)
			}
		}
		if len(referencedIDs) > 0 {
			if linkedPrs, err := issues_model.Issues(ctx, &issues_model.IssuesOptions{
				IssueIDs: referencedIDs,
				IsPull:   optional.Some(true),
			}); err == nil {
				linkedPRsMap[issue.ID] = linkedPrs
			} else {
				log.Error("LoadLinkedPRs for issue %d: %v", issue.ID, err)
			}
		}
	}
	return linkedPRsMap
}

func IssuePane(ctx *context.Context) {
	prepareIssueViewData(ctx)
	if ctx.Written() {
		return
	}
	ctx.Data["IsPaneMode"] = true
	ctx.HTML(http.StatusOK, "repo/issue/view_pane")
}

func PullPaneFiles(ctx *context.Context) {
	preparePullFilesData(ctx, "", "", false, false)
	if ctx.Written() {
		return
	}
	ctx.Data["IsPaneMode"] = true
	ctx.HTML(http.StatusOK, "repo/issue/diff_pane")
}

func PullPaneCommits(ctx *context.Context) {
	preparePullCommitsData(ctx)
	if ctx.Written() {
		return
	}
	ctx.Data["IsPaneMode"] = true
	ctx.HTML(http.StatusOK, "repo/issue/commits_pane")
}

func PullPaneTabData(ctx *context.Context) {
	issue, ok := getPullInfo(ctx)
	if !ok {
		return
	}
	pull := issue.PullRequest

	var prInfo *git.CompareInfo
	if pull.HasMerged {
		prInfo = PrepareMergedViewPullInfo(ctx, issue)
	} else {
		prInfo = PrepareViewPullInfo(ctx, issue)
	}

	numCommits := 0
	numFiles := 0
	if prInfo != nil {
		numCommits = len(prInfo.Commits)
		numFiles = prInfo.NumFiles
	}

	ctx.JSON(http.StatusOK, map[string]int{
		"comments": issue.NumComments,
		"commits":  numCommits,
		"files":    numFiles,
	})
}
