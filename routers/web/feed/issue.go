// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"time"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/services/context"

	"github.com/gorilla/feeds"
)

func isFeedRelevant(commentType issues_model.CommentType) bool {
	switch commentType {
	case issues_model.CommentTypeComment:
		return true
	case issues_model.CommentTypeReopen:
		return true
	case issues_model.CommentTypeClose:
		return true
	case issues_model.CommentTypeIssueRef:
		return true
	case issues_model.CommentTypeCommitRef:
		return true
	case issues_model.CommentTypeCommentRef:
		return true
	case issues_model.CommentTypePullRef:
		return true
	case issues_model.CommentTypeLabel:
		return true
	case issues_model.CommentTypeMilestone:
		return true
	case issues_model.CommentTypeAssignees:
		return true
	case issues_model.CommentTypeChangeTitle:
		return true
	case issues_model.CommentTypeAddedDeadline:
		return true
	case issues_model.CommentTypeModifiedDeadline:
		return true
	case issues_model.CommentTypeRemovedDeadline:
		return true
	case issues_model.CommentTypeAddDependency:
		return true
	case issues_model.CommentTypeRemoveDependency:
		return true
	case issues_model.CommentTypeCode:
		return true
	case issues_model.CommentTypeReview:
		return true
	case issues_model.CommentTypeLock:
		return true
	case issues_model.CommentTypeUnlock:
		return true
	case issues_model.CommentTypeReviewRequest:
		return true
	case issues_model.CommentTypeMergePull:
		return true
	case issues_model.CommentTypeProject:
		return true
	case issues_model.CommentTypeDismissReview:
		return true
	case issues_model.CommentTypePRScheduledToAutoMerge:
		return true
	case issues_model.CommentTypePRUnScheduledToAutoMerge:
		return true
	case issues_model.CommentTypePin:
		return true
	case issues_model.CommentTypeUnpin:
		return true
	default:
		return false
	}
}

// shows an issue on the repo as RSS / Atom feed
func ShowIssueFeed(ctx *context.Context, issue *issues_model.Issue, formatType string) {
	title := issue.Title
	link := &feeds.Link{Href: issue.HTMLURL()}

	feed := &feeds.Feed{
		Title:       title,
		Link:        link,
		Description: issue.Content,
		Created:     time.Now(),
	}

	issueComments := make([]*issues_model.Comment, 0, len(issue.Comments))
	for _, issueComment := range issue.Comments {
		if isFeedRelevant(issueComment.Type) {
			issueComments = append(issueComments, issueComment)
		}
	}

	var err error
	feed.Items, err = issueCommentsToFeedItems(ctx, issueComments)
	if err != nil {
		ctx.ServerError("issueCommentsToFeedItems", err)
		return
	}

	writeFeed(ctx, feed, formatType)
}
