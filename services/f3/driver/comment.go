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
	permissions_comment "forgejo.org/services/permissions/comment"

	"code.forgejo.org/f3/gof3/v3/f3"
	"code.forgejo.org/f3/gof3/v3/f3/markdown"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &comment{}

type comment struct {
	common

	forgejoComment *issues_model.Comment
}

func (o *comment) SetNative(comment any) {
	o.forgejoComment = comment.(*issues_model.Comment)
}

func (o *comment) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoComment.ID)
}

func (o *comment) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *comment) ToFormat() f3.Interface {
	if o.forgejoComment == nil {
		return o.NewFormat()
	}
	return (&f3.Comment{
		Common:   f3.NewCommon(fmt.Sprintf("%d", o.forgejoComment.ID)),
		PosterID: f3_tree.NewUserReference(f3_util.ToString(o.forgejoComment.Poster.ID)),
		Content:  markdown.NewContent().Set(o.forgejoComment.Content),
		Created:  o.forgejoComment.CreatedUnix.AsTime(),
		Updated:  o.forgejoComment.UpdatedUnix.AsTime(),
	}).Init()
}

func (o *comment) FromFormat(content f3.Interface) {
	comment := content.(*f3.Comment)

	o.forgejoComment = &issues_model.Comment{
		ID:       f3_util.ParseInt(comment.GetID()),
		Type:     issues_model.CommentTypeComment,
		PosterID: comment.PosterID.GetIDAsInt(),
		Poster: &user_model.User{
			ID: comment.PosterID.GetIDAsInt(),
		},
		Content:     comment.Content.Get(),
		CreatedUnix: timeutil.TimeStamp(comment.Created.Unix()),
		UpdatedUnix: timeutil.TimeStamp(comment.Updated.Unix()),
	}
}

func (o *comment) loadPoster(ctx context.Context) {
	if err := o.forgejoComment.LoadPoster(ctx); err != nil {
		panic(fmt.Errorf("LoadPoster(%+v): %w", o.forgejoComment, err))
	}
}

func (o *comment) loadIssue(ctx context.Context) {
	if err := o.forgejoComment.LoadIssue(ctx); err != nil {
		panic(fmt.Errorf("LoadIssue(%+v): %w", o.forgejoComment, err))
	}
}

func (o *comment) loadRepo(ctx context.Context) {
	o.loadIssue(ctx)
	if err := o.forgejoComment.Issue.LoadRepo(ctx); err != nil {
		panic(fmt.Errorf("LoadRepo(%+v): %w", o.forgejoComment.Issue, err))
	}
}

func (o *comment) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	comment, err := issues_model.GetCommentByID(ctx, id)
	if issues_model.IsErrCommentNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("comment %v %w", id, err))
	}
	o.forgejoComment = comment
	o.loadPoster(ctx)

	permissionsCheck(ctx, permissions_comment.Get(f3_tree.GetProjectID(node), o.getIssueOrPullRequestAbsoluteID(ctx, node), o.forgejoComment))

	return true
}

func (o *comment) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoComment.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoComment.ID).Cols("content", "updated").NoAutoTime().Update(o.forgejoComment); err != nil {
		panic(fmt.Errorf("UpdateCommentCols: %v %v", o.forgejoComment, err))
	}
}

func (o *comment) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()

	issueIndex := f3_tree.GetCommentableID(node)
	repositoryID := f3_tree.GetProjectID(node)

	issue, err := issues_model.GetIssueByIndex(ctx, repositoryID, issueIndex)
	if issues_model.IsErrIssueNotExist(err) {
		panic(fmt.Errorf("issue index %d not found in repository id %d", issueIndex, repositoryID))
	}
	o.forgejoComment.IssueID = issue.ID

	sess := db.GetEngine(ctx)

	if _, err := sess.NoAutoTime().Insert(o.forgejoComment); err != nil {
		panic(err)
	}
	o.Trace("comment created %d", o.forgejoComment.ID)
	if o.sendNotifications(ctx) {
		o.loadPoster(ctx)
		o.loadIssue(ctx)
		o.loadRepo(ctx)
		notify_service.CreateIssueComment(ctx, o.forgejoComment.Poster, o.forgejoComment.Issue.Repo, o.forgejoComment.Issue, o.forgejoComment, nil)
	}
	return f3_id.NewNodeID(o.forgejoComment.ID)
}

func (o *comment) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := issues_model.DeleteComment(ctx, o.forgejoComment); err != nil {
		panic(err)
	}
}

func newComment() generic.NodeDriverInterface {
	return &comment{}
}
