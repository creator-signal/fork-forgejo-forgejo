// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/timeutil"
	notify_service "forgejo.org/services/notify"

	"code.forgejo.org/f3/gof3/v3/f3"
	"code.forgejo.org/f3/gof3/v3/f3/markdown"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &review{}

type review struct {
	common

	forgejoReview *issues_model.Review
}

func (o *review) SetNative(review any) {
	o.forgejoReview = review.(*issues_model.Review)
}

func (o *review) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoReview.ID)
}

func (o *review) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *review) ToFormat() f3.Interface {
	if o.forgejoReview == nil {
		return o.NewFormat()
	}

	review := (&f3.Review{
		Common:     f3.NewCommon(o.GetNativeID()),
		ReviewerID: f3_tree.NewUserReference(f3_util.ToString(o.forgejoReview.ReviewerID)),
		Official:   o.forgejoReview.Official,
		CommitID:   o.forgejoReview.CommitID,
		Content:    markdown.NewContent().Set(o.forgejoReview.Content),
		CreatedAt:  o.forgejoReview.CreatedUnix.AsTime(),
	}).Init()

	switch o.forgejoReview.Type {
	case issues_model.ReviewTypeApprove:
		review.State = f3.ReviewStateApproved
	case issues_model.ReviewTypeReject:
		review.State = f3.ReviewStateChangesRequested
	case issues_model.ReviewTypeComment:
		review.State = f3.ReviewStateCommented
	case issues_model.ReviewTypePending:
		review.State = f3.ReviewStatePending
	case issues_model.ReviewTypeRequest:
		review.State = f3.ReviewStateRequestReview
	default:
		review.State = f3.ReviewStateUnknown
	}

	if o.forgejoReview.Reviewer != nil {
		review.ReviewerID = f3_tree.NewUserReference(f3_util.ToString(o.forgejoReview.Reviewer.ID))
	}

	return review
}

func (o *review) FromFormat(content f3.Interface) {
	review := content.(*f3.Review)

	o.forgejoReview = &issues_model.Review{
		ID:         f3_util.ParseInt(review.GetID()),
		ReviewerID: review.ReviewerID.GetIDAsInt(),
		Reviewer: &user_model.User{
			ID: review.ReviewerID.GetIDAsInt(),
		},
		Official:    review.Official,
		CommitID:    review.CommitID,
		Content:     review.Content.Get(),
		CreatedUnix: timeutil.TimeStamp(review.CreatedAt.Unix()),
	}

	switch review.State {
	case f3.ReviewStateApproved:
		o.forgejoReview.Type = issues_model.ReviewTypeApprove
	case f3.ReviewStateChangesRequested:
		o.forgejoReview.Type = issues_model.ReviewTypeReject
	case f3.ReviewStateCommented:
		o.forgejoReview.Type = issues_model.ReviewTypeComment
	case f3.ReviewStatePending:
		o.forgejoReview.Type = issues_model.ReviewTypePending
	case f3.ReviewStateRequestReview:
		o.forgejoReview.Type = issues_model.ReviewTypeRequest
	default:
		o.forgejoReview.Type = issues_model.ReviewTypeUnknown
	}
}

func (o *review) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	review, err := issues_model.GetReviewByID(ctx, id)
	if issues_model.IsErrReviewNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("review %v %w", id, err))
	}
	if err := review.LoadReviewer(ctx); err != nil {
		panic(fmt.Errorf("LoadReviewer %v %w", *review, err))
	}
	o.forgejoReview = review
	return true
}

func (o *review) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoReview.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoReview.ID).Cols("content").Update(o.forgejoReview); err != nil {
		panic(fmt.Errorf("UpdateReviewCols: %v %v", o.forgejoReview, err))
	}
}

func reviewTypeToCommentType(reviewType issues_model.ReviewType) issues_model.CommentType {
	switch reviewType {
	case issues_model.ReviewTypeComment:
		return issues_model.CommentTypeReview
	case issues_model.ReviewTypeRequest:
		return issues_model.CommentTypeReviewRequest
	default:
		return issues_model.CommentTypeReview
	}
}

func (o *review) Put(ctx context.Context) f3_id.NodeID {
	project := f3_tree.GetProjectID(o.GetNode())
	pullRequest := f3_tree.GetPullRequestID(o.GetNode())

	issue, err := issues_model.GetIssueByIndex(ctx, project, pullRequest)
	if err != nil {
		panic(fmt.Errorf("GetIssueByIndex %v", err))
	}
	o.forgejoReview.Issue = issue
	o.forgejoReview.IssueID = issue.ID

	sess := db.GetEngine(ctx)

	if _, err := sess.NoAutoTime().Insert(o.forgejoReview); err != nil {
		panic(err)
	}

	commentType := reviewTypeToCommentType(o.forgejoReview.Type)
	comment := &issues_model.Comment{
		Type:     commentType,
		IssueID:  issue.ID,
		ReviewID: o.forgejoReview.ID,
		PosterID: o.forgejoReview.ReviewerID,
		Poster: &user_model.User{
			ID: o.forgejoReview.ReviewerID,
		},
		Content:     o.forgejoReview.Content,
		CreatedUnix: o.forgejoReview.CreatedUnix,
		UpdatedUnix: o.forgejoReview.UpdatedUnix,
	}

	if _, err := sess.NoAutoTime().Insert(comment); err != nil {
		panic(err)
	}

	o.Trace("review created %d for pull request %d", o.forgejoReview.ID, o.forgejoReview.IssueID)
	if o.sendNotifications(ctx) {
		if err := o.forgejoReview.Issue.LoadPullRequest(ctx); err != nil {
			panic(err)
		}
		notify_service.PullRequestReview(ctx, o.forgejoReview.Issue.PullRequest, o.forgejoReview, comment, nil)
	}
	return f3_id.NewNodeID(o.forgejoReview.ID)
}

func (o *review) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	project := f3_tree.GetProjectID(o.GetNode())
	pullRequest := f3_tree.GetPullRequestID(o.GetNode())

	issue, err := issues_model.GetIssueByIndex(ctx, project, pullRequest)
	if err != nil {
		panic(fmt.Errorf("GetIssueByIndex %v", err))
	}
	o.forgejoReview.IssueID = issue.ID

	if o.forgejoReview.Type == issues_model.ReviewTypeRequest {
		if _, err := db.DeleteByBean(ctx, issues_model.Comment{ReviewID: o.forgejoReview.ID}); err != nil {
			panic(err)
		}
		if _, err := db.DeleteByID[issues_model.Review](ctx, o.forgejoReview.ID); err != nil {
			panic(err)
		}
	} else {
		if err := issues_model.DeleteReview(ctx, o.forgejoReview); err != nil {
			panic(err)
		}
	}
}

func newReview() generic.NodeDriverInterface {
	return &review{}
}
