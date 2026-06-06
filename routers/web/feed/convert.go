// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	activities_model "forgejo.org/models/activities"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/markup"
	"forgejo.org/modules/markup/markdown"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/templates"
	"forgejo.org/modules/util"
	"forgejo.org/services/context"

	"github.com/gorilla/feeds"
	"github.com/inbucket/html2text"
)

func toBranchLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/src/branch/" + util.PathEscapeSegments(act.GetBranch())
}

func toTagLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/src/tag/" + util.PathEscapeSegments(act.GetTag())
}

func toIssueLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/issues/" + url.PathEscape(act.GetIssueInfos()[0])
}

func toPullLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/pulls/" + url.PathEscape(act.GetIssueInfos()[0])
}

func toSrcLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/src/" + util.PathEscapeSegments(act.GetBranch())
}

func toReleaseLink(ctx *context.Context, act *activities_model.Action) string {
	return act.GetRepoAbsoluteLink(ctx) + "/releases/tag/" + util.PathEscapeSegments(act.GetBranch())
}

// renderMarkdown creates a minimal markdown render context from an action.
// If rendering fails, the original markdown text is returned
func renderMarkdown(ctx *context.Context, act *activities_model.Action, content string) template.HTML {
	markdownCtx := &markup.RenderContext{
		Ctx: ctx,
		Links: markup.Links{
			Base: act.GetRepoLink(ctx),
		},
		Type: markdown.MarkupName,
		Metas: map[string]string{
			"user": act.GetRepoUserName(ctx),
			"repo": act.GetRepoName(ctx),
		},
	}
	markdown, err := markdown.RenderString(markdownCtx, content)
	if err != nil {
		return templates.SanitizeHTML(content) // old code did so: use SanitizeHTML to render in tmpl
	}
	return markdown
}

// feedActionsToFeedItems convert gitea's Action feed to feeds Item
func feedActionsToFeedItems(ctx *context.Context, actions activities_model.ActionList) (items []*feeds.Item, err error) {
	for _, act := range actions {
		act.LoadActUser(ctx)

		// TODO: the code seems quite strange (maybe not right)
		// sometimes it uses text content but sometimes it uses HTML content
		// it should clearly defines which kind of content it should use for the feed items: plan text or rich HTML
		var title, desc string
		var content template.HTML

		link := &feeds.Link{Href: act.GetCommentHTMLURL(ctx)}

		// title
		title = act.ActUser.GetDisplayName() + " "
		var titleExtra template.HTML
		switch act.OpType {
		case activities_model.ActionCreateRepo:
			titleExtra = ctx.Locale.Tr("action.create_repo", act.GetRepoAbsoluteLink(ctx), act.ShortRepoPath(ctx))
			link.Href = act.GetRepoAbsoluteLink(ctx)
		case activities_model.ActionRenameRepo:
			titleExtra = ctx.Locale.Tr("action.rename_repo", act.GetContent(), act.GetRepoAbsoluteLink(ctx), act.ShortRepoPath(ctx))
			link.Href = act.GetRepoAbsoluteLink(ctx)
		case activities_model.ActionCommitRepo:
			link.Href = toBranchLink(ctx, act)
			if len(act.Content) != 0 {
				titleExtra = ctx.Locale.Tr("action.commit_repo", act.GetRepoAbsoluteLink(ctx), link.Href, act.GetBranch(), act.ShortRepoPath(ctx))
			} else {
				titleExtra = ctx.Locale.Tr("action.create_branch", act.GetRepoAbsoluteLink(ctx), link.Href, act.GetBranch(), act.ShortRepoPath(ctx))
			}
		case activities_model.ActionCreateIssue:
			link.Href = toIssueLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.create_issue", link.Href, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionCreatePullRequest:
			link.Href = toPullLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.create_pull_request", link.Href, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionTransferRepo:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.transfer_repo", act.GetContent(), act.GetRepoAbsoluteLink(ctx), act.ShortRepoPath(ctx))
		case activities_model.ActionPushTag:
			link.Href = toTagLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.push_tag", act.GetRepoAbsoluteLink(ctx), link.Href, act.GetTag(), act.ShortRepoPath(ctx))
		case activities_model.ActionCommentIssue:
			issueLink := toIssueLink(ctx, act)
			if link.Href == "#" {
				link.Href = issueLink
			}
			titleExtra = ctx.Locale.Tr("action.comment_issue", issueLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionMergePullRequest:
			pullLink := toPullLink(ctx, act)
			if link.Href == "#" {
				link.Href = pullLink
			}
			titleExtra = ctx.Locale.Tr("action.merge_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionAutoMergePullRequest:
			pullLink := toPullLink(ctx, act)
			if link.Href == "#" {
				link.Href = pullLink
			}
			titleExtra = ctx.Locale.Tr("action.auto_merge_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionCloseIssue:
			issueLink := toIssueLink(ctx, act)
			if link.Href == "#" {
				link.Href = issueLink
			}
			titleExtra = ctx.Locale.Tr("action.close_issue", issueLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionReopenIssue:
			issueLink := toIssueLink(ctx, act)
			if link.Href == "#" {
				link.Href = issueLink
			}
			titleExtra = ctx.Locale.Tr("action.reopen_issue", issueLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionClosePullRequest:
			pullLink := toPullLink(ctx, act)
			if link.Href == "#" {
				link.Href = pullLink
			}
			titleExtra = ctx.Locale.Tr("action.close_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionReopenPullRequest:
			pullLink := toPullLink(ctx, act)
			if link.Href == "#" {
				link.Href = pullLink
			}
			titleExtra = ctx.Locale.Tr("action.reopen_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionDeleteTag:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.delete_tag", act.GetRepoAbsoluteLink(ctx), act.GetTag(), act.ShortRepoPath(ctx))
		case activities_model.ActionDeleteBranch:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.delete_branch", act.GetRepoAbsoluteLink(ctx), html.EscapeString(act.GetBranch()), act.ShortRepoPath(ctx))
		case activities_model.ActionMirrorSyncPush:
			srcLink := toSrcLink(ctx, act)
			if link.Href == "#" {
				link.Href = srcLink
			}
			titleExtra = ctx.Locale.Tr("action.mirror_sync_push", act.GetRepoAbsoluteLink(ctx), srcLink, act.GetBranch(), act.ShortRepoPath(ctx))
		case activities_model.ActionMirrorSyncCreate:
			srcLink := toSrcLink(ctx, act)
			if link.Href == "#" {
				link.Href = srcLink
			}
			titleExtra = ctx.Locale.Tr("action.mirror_sync_create", act.GetRepoAbsoluteLink(ctx), srcLink, act.GetBranch(), act.ShortRepoPath(ctx))
		case activities_model.ActionMirrorSyncDelete:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.mirror_sync_delete", act.GetRepoAbsoluteLink(ctx), act.GetBranch(), act.ShortRepoPath(ctx))
		case activities_model.ActionApprovePullRequest:
			pullLink := toPullLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.approve_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionRejectPullRequest:
			pullLink := toPullLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.reject_pull_request", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionCommentPull:
			pullLink := toPullLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.comment_pull", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx))
		case activities_model.ActionPublishRelease:
			releaseLink := toReleaseLink(ctx, act)
			if link.Href == "#" {
				link.Href = releaseLink
			}
			titleExtra = ctx.Locale.Tr("action.publish_release", act.GetRepoAbsoluteLink(ctx), releaseLink, act.ShortRepoPath(ctx), act.Content)
		case activities_model.ActionPullReviewDismissed:
			pullLink := toPullLink(ctx, act)
			titleExtra = ctx.Locale.Tr("action.review_dismissed", pullLink, act.GetIssueInfos()[0], act.ShortRepoPath(ctx), act.GetIssueInfos()[1])
		case activities_model.ActionStarRepo:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.starred_repo", act.GetRepoAbsoluteLink(ctx), act.GetRepoPath(ctx))
		case activities_model.ActionWatchRepo:
			link.Href = act.GetRepoAbsoluteLink(ctx)
			titleExtra = ctx.Locale.Tr("action.watched_repo", act.GetRepoAbsoluteLink(ctx), act.GetRepoPath(ctx))
		default:
			return nil, fmt.Errorf("unknown action type: %v", act.OpType)
		}

		// description & content
		{
			switch act.OpType {
			case activities_model.ActionCommitRepo, activities_model.ActionMirrorSyncPush:
				push := templates.ActionContent2Commits(ctx, act)

				for _, commit := range push.Commits {
					if len(desc) != 0 {
						desc += "\n\n"
					}
					desc += fmt.Sprintf("<a href=\"%s\">%s</a>\n%s",
						html.EscapeString(fmt.Sprintf("%s/commit/%s", act.GetRepoAbsoluteLink(ctx), commit.Sha1)),
						commit.Sha1,
						templates.RenderCommitMessage(ctx, commit.Message, nil),
					)
				}

				if push.Len > 1 {
					link = &feeds.Link{Href: setting.AppURL + push.CompareURL}
				} else if push.Len == 1 {
					link = &feeds.Link{Href: fmt.Sprintf("%s/commit/%s", act.GetRepoAbsoluteLink(ctx), push.Commits[0].Sha1)}
				}

			case activities_model.ActionCreateIssue, activities_model.ActionCreatePullRequest:
				desc = strings.Join(act.GetIssueInfos(), "#")
				content = renderMarkdown(ctx, act, act.GetIssueContent(ctx))
			case activities_model.ActionCommentIssue, activities_model.ActionApprovePullRequest, activities_model.ActionRejectPullRequest, activities_model.ActionCommentPull:
				desc = act.GetIssueTitle(ctx)
				comment := act.GetIssueInfos()[1]
				if strings.HasSuffix(comment, "…") {
					// Comment was truncated get the full content from the database.
					// This truncation is done in `NotifyCreateIssueComment`.
					commentModel, err := issues_model.GetCommentByID(ctx, act.CommentID)
					if err != nil {
						log.Error("Couldn't get comment[%d] for RSS feed: %v", act.CommentID, err)
					} else {
						comment = commentModel.Content
					}
				}
				if len(comment) != 0 {
					desc += "\n\n" + string(renderMarkdown(ctx, act, comment))
				}
			case activities_model.ActionMergePullRequest, activities_model.ActionAutoMergePullRequest:
				desc = act.GetIssueInfos()[1]
			case activities_model.ActionCloseIssue, activities_model.ActionReopenIssue, activities_model.ActionClosePullRequest, activities_model.ActionReopenPullRequest:
				desc = act.GetIssueTitle(ctx)
			case activities_model.ActionPullReviewDismissed:
				desc = ctx.Locale.TrString("action.review_dismissed_reason") + "\n\n" + act.GetIssueInfos()[2]
			}
		}
		if len(content) == 0 {
			content = templates.SanitizeHTML(desc)
		}

		// It's a common practice for feed generators to use plain text titles.
		// See https://codeberg.org/forgejo/forgejo/pulls/1595
		plainTitle, err := html2text.FromString(title+" "+string(titleExtra), html2text.Options{OmitLinks: true})
		if err != nil {
			return nil, err
		}

		items = append(items, &feeds.Item{
			Title:       plainTitle,
			Link:        link,
			Description: desc,
			IsPermaLink: "false",
			Author: &feeds.Author{
				Name:  act.ActUser.GetDisplayName(),
				Email: act.ActUser.GetEmail(),
			},
			Id:      fmt.Sprintf("%v: %v", strconv.FormatInt(act.ID, 10), link.Href),
			Created: act.CreatedUnix.AsTime(),
			Content: string(content),
		})
	}
	return items, err
}

// GetFeedType return if it is a feed request and altered name and feed type.
func GetFeedType(name string, req *http.Request) (bool, string, string) {
	if strings.HasSuffix(name, ".rss") ||
		strings.Contains(req.Header.Get("Accept"), "application/rss+xml") {
		return true, strings.TrimSuffix(name, ".rss"), "rss"
	}

	if strings.HasSuffix(name, ".atom") ||
		strings.Contains(req.Header.Get("Accept"), "application/atom+xml") {
		return true, strings.TrimSuffix(name, ".atom"), "atom"
	}

	return false, name, ""
}

func issueCommentsToFeedItems(ctx *context.Context, comments issues_model.CommentList) (items []*feeds.Item, err error) {
	items = make([]*feeds.Item, 0, len(comments))
	users := make(map[int64]*user_model.User)
	for _, comment := range comments {
		user, userCached := users[comment.PosterID]
		if !userCached {
			user, err = user_model.GetUserByID(ctx, comment.PosterID)
			if err != nil {
				return items, err
			}
			users[user.ID] = user
		}
		var title string
		switch comment.Type {
		case issues_model.CommentTypeComment:
			title = ctx.Locale.TrString("repo.comment_type.comment")
		case issues_model.CommentTypeReopen:
			title = ctx.Locale.TrString("repo.comment_type.re_open")
		case issues_model.CommentTypeClose:
			title = ctx.Locale.TrString("repo.comment_type.closed")
		case issues_model.CommentTypeIssueRef:
			title = ctx.Locale.TrString("repo.comment_type.issue_reference")
		case issues_model.CommentTypeCommitRef:
			title = ctx.Locale.TrString("repo.comment_type.commit_reference")
		case issues_model.CommentTypeCommentRef:
			title = ctx.Locale.TrString("repo.comment_type.comment_reference")
		case issues_model.CommentTypePullRef:
			title = ctx.Locale.TrString("repo.comment_type.pr_reference")
		case issues_model.CommentTypeLabel:
			title = ctx.Locale.TrString("repo.comment_type.label_changed")
		case issues_model.CommentTypeMilestone:
			title = ctx.Locale.TrString("repo.comment_type.milestone_changed")
		case issues_model.CommentTypeAssignees:
			title = ctx.Locale.TrString("repo.comment_type.assignees_changed")
		case issues_model.CommentTypeChangeTitle:
			title = ctx.Locale.TrString("repo.comment_type.title_changed")
		case issues_model.CommentTypeDeleteBranch:
			title = ctx.Locale.TrString("repo.comment_type.branch_deleted")
		case issues_model.CommentTypeStartTracking:
			title = ctx.Locale.TrString("repo.comment_type.time_tracking_started")
		case issues_model.CommentTypeStopTracking:
			title = ctx.Locale.TrString("repo.comment_type.time_tracking_stopped")
		case issues_model.CommentTypeAddTimeManual:
			title = ctx.Locale.TrString("repo.comment_type.time_added_manually")
		case issues_model.CommentTypeCancelTracking:
			title = ctx.Locale.TrString("repo.comment_type.tracking_canceled")
		case issues_model.CommentTypeAddedDeadline:
			title = ctx.Locale.TrString("repo.comment_type.deadline_added")
		case issues_model.CommentTypeModifiedDeadline:
			title = ctx.Locale.TrString("repo.comment_type.deadline_modified")
		case issues_model.CommentTypeRemovedDeadline:
			title = ctx.Locale.TrString("repo.comment_type.deadline_removed")
		case issues_model.CommentTypeAddDependency:
			title = ctx.Locale.TrString("repo.comment_type.dependency_added")
		case issues_model.CommentTypeRemoveDependency:
			title = ctx.Locale.TrString("repo.comment_type.dependency_removed")
		case issues_model.CommentTypeCode:
			title = ctx.Locale.TrString("repo.comment_type.code_commented")
		case issues_model.CommentTypeReview:
			title = ctx.Locale.TrString("repo.comment_type.code_reviewed")
		case issues_model.CommentTypeLock:
			title = ctx.Locale.TrString("repo.comment_type.issue_locked")
		case issues_model.CommentTypeUnlock:
			title = ctx.Locale.TrString("repo.comment_type.issue_unlocked")
		case issues_model.CommentTypeChangeTargetBranch:
			title = ctx.Locale.TrString("repo.comment_type.change_target_branch")
		case issues_model.CommentTypeDeleteTimeManual:
			title = ctx.Locale.TrString("repo.comment_type.time_deleted_manually")
		case issues_model.CommentTypeReviewRequest:
			title = ctx.Locale.TrString("repo.comment_type.review_request")
		case issues_model.CommentTypeMergePull:
			title = ctx.Locale.TrString("repo.comment_type.pr_merged")
		case issues_model.CommentTypePullRequestPush:
			title = ctx.Locale.TrString("repo.comment_type.pr_pushed")
		case issues_model.CommentTypeProject:
			title = ctx.Locale.TrString("repo.comment_type.project_changed")
		case issues_model.CommentTypeProjectColumn:
			title = ctx.Locale.TrString("repo.comment_type.project_column_changed")
		case issues_model.CommentTypeDismissReview:
			title = ctx.Locale.TrString("repo.comment_type.review_dismissed")
		case issues_model.CommentTypeChangeIssueRef:
			title = ctx.Locale.TrString("repo.comment_type.change_issue_ref")
		case issues_model.CommentTypePRScheduledToAutoMerge:
			title = ctx.Locale.TrString("repo.comment_type.")
		case issues_model.CommentTypePRUnScheduledToAutoMerge:
			title = ctx.Locale.TrString("repo.comment_type.pr_scheduled_to_auto_merge")
		case issues_model.CommentTypePin:
			title = ctx.Locale.TrString("repo.comment_type.issue_pinned")
		case issues_model.CommentTypeUnpin:
			title = ctx.Locale.TrString("repo.comment_type.issue_unpinned")
		case issues_model.CommentTypeAggregator:
			title = ctx.Locale.TrString("repo.comment_type.aggregator")
		}
		item := &feeds.Item{
			Id:          fmt.Sprintf("%v: %v", strconv.FormatInt(comment.ID, 10), comment.IssueURL(ctx)),
			Title:       title,
			Link:        &feeds.Link{Href: comment.IssueURL(ctx)},
			Description: comment.Content,
			Created:     comment.CreatedUnix.AsTime(),
			Updated:     comment.UpdatedUnix.AsTime(),
			Author:      &feeds.Author{Name: user.DisplayName(), Email: user.GetEmail()},
		}
		items = append(items, item)
	}
	return items, nil
}

// feedActionsToFeedItems convert repository releases into feed items.
func releasesToFeedItems(ctx *context.Context, releases repo_model.ReleaseList) (items []*feeds.Item, err error) {
	if err := releases.LoadAttributes(ctx); err != nil {
		return nil, err
	}

	composeCache := make(map[int64]map[string]string)
	for _, rel := range releases {
		var title string
		var content template.HTML

		if rel.IsTag {
			title = rel.TagName
		} else {
			title = rel.Title
		}

		metas, ok := composeCache[rel.RepoID]
		if !ok {
			metas = rel.Repo.ComposeMetas(ctx)
			composeCache[rel.RepoID] = metas
		}

		link := &feeds.Link{Href: rel.HTMLURL()}
		content, err = markdown.RenderString(&markup.RenderContext{
			Ctx: ctx,
			Links: markup.Links{
				Base: rel.Repo.Link(),
			},
			Metas: metas,
		}, rel.Note)
		if err != nil {
			return nil, err
		}

		items = append(items, &feeds.Item{
			Title:   title,
			Link:    link,
			Created: rel.CreatedUnix.AsTime(),
			Author: &feeds.Author{
				Name:  rel.Publisher.GetDisplayName(),
				Email: rel.Publisher.GetEmail(),
			},
			Id:      fmt.Sprintf("%v: %v", strconv.FormatInt(rel.ID, 10), link.Href),
			Content: string(content),
		})
	}

	return items, err
}
