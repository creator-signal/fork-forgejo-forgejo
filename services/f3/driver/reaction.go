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

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &reaction{}

type reaction struct {
	common

	forgejoReaction *issues_model.Reaction
}

func (o *reaction) SetNative(reaction any) {
	o.forgejoReaction = reaction.(*issues_model.Reaction)
}

func (o *reaction) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoReaction.ID)
}

func (o *reaction) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *reaction) ToFormat() f3.Interface {
	if o.forgejoReaction == nil {
		return o.NewFormat()
	}
	return (&f3.Reaction{
		Common:  f3.NewCommon(fmt.Sprintf("%d", o.forgejoReaction.ID)),
		UserID:  f3_tree.NewUserReference(f3_util.ToString(o.forgejoReaction.User.ID)),
		Content: o.forgejoReaction.Type,
	}).Init()
}

func (o *reaction) FromFormat(content f3.Interface) {
	reaction := content.(*f3.Reaction)

	o.forgejoReaction = &issues_model.Reaction{
		ID:     f3_util.ParseInt(reaction.GetID()),
		UserID: reaction.UserID.GetIDAsInt(),
		User: &user_model.User{
			ID: reaction.UserID.GetIDAsInt(),
		},
		Type: reaction.Content,
	}
}

func (o *reaction) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	var reaction issues_model.Reaction
	if has, err := db.GetEngine(ctx).Where("ID = ?", id).Get(&reaction); err != nil {
		panic(fmt.Errorf("reaction %v - %w", id, err))
	} else if !has {
		return false
	}
	o.forgejoReaction = &reaction
	if _, err := o.forgejoReaction.LoadUser(ctx); err != nil {
		panic(fmt.Errorf("LoadUser %v %w", *o.forgejoReaction, err))
	}
	return true
}

func (o *reaction) Patch(ctx context.Context) {
	o.Trace("%d", o.forgejoReaction.ID)
	if _, err := db.GetEngine(ctx).ID(o.forgejoReaction.ID).Cols("type").Update(o.forgejoReaction); err != nil {
		panic(fmt.Errorf("UpdateReactionCols: %v %v", o.forgejoReaction, err))
	}
}

func (o *reaction) Put(ctx context.Context) f3_id.NodeID {
	sess := db.GetEngine(ctx)

	node := o.GetNode()
	setCommentID := func(reactionableID int64) {
		o.forgejoReaction.CommentID = reactionableID
	}
	reactionable := f3_tree.GetReactionable(node)
	reactionableID := f3_tree.GetReactionableID(node)

	o.Trace("creating %s reaction %s", reactionable.GetKind(), o.forgejoReaction.Type)

	o.forgejoReaction.IssueID = o.getIssueOrPullRequestAbsoluteID(ctx, node)

	switch reactionable.GetKind() {
	case f3_kind.KindIssue, f3_kind.KindPullRequest:
		break
	case f3_kind.KindComment:
		setCommentID(reactionableID)
	case f3_kind.KindReviewComment:
		setCommentID(reactionableID)
	case f3_kind.KindReview:
		break
	default:
		panic(fmt.Errorf("unexpected kind %v", reactionable.GetKind()))
	}

	if _, err := sess.Insert(o.forgejoReaction); err != nil {
		panic(fmt.Errorf("%+v: %v", o.forgejoReaction, err))
	}
	o.Trace("reaction created %+v", o.forgejoReaction)
	return f3_id.NewNodeID(o.forgejoReaction.ID)
}

func (o *reaction) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	sess := db.GetEngine(ctx)
	if _, err := sess.Delete(o.forgejoReaction); err != nil {
		panic(err)
	}
}

func newReaction() generic.NodeDriverInterface {
	return &reaction{}
}
