// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"

	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	"xorm.io/builder"
)

type reactions struct {
	container
}

func (o *reactions) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	pageSize := o.getPageSize()

	const issueField = "reaction.issue_id"
	andIssue := func(cond builder.Cond) builder.Cond {
		project := f3_tree.GetProjectID(node)
		commentableID := f3_tree.GetCommentableID(o.GetNode())
		issue, err := issues_model.GetIssueByIndex(ctx, project, commentableID)
		if err != nil {
			panic(fmt.Errorf("GetIssueByIndex %v %w", commentableID, err))
		}
		return cond.And(builder.Eq{issueField: issue.ID})
	}

	const commentField = "reaction.comment_id"
	andComment := func(cond builder.Cond, reactionableID int64) builder.Cond {
		return cond.And(builder.Eq{commentField: reactionableID})
	}

	andZeroField := func(cond builder.Cond, field string) builder.Cond {
		return cond.And(builder.Eq{field: 0})
	}

	reactionable := f3_tree.GetReactionable(node)
	reactionableID := f3_tree.GetReactionableID(node)

	o.Trace("%s %d", reactionable.GetKind(), reactionableID)
	sess := db.GetEngine(ctx)
	cond := builder.NewCond()
	switch reactionable.GetKind() {
	case f3_kind.KindIssue, f3_kind.KindPullRequest:
		cond = andIssue(cond)
		cond = andZeroField(cond, commentField)
	case f3_kind.KindComment:
		cond = andZeroField(cond, issueField)
		cond = andComment(cond, reactionableID)
	case f3_kind.KindReviewComment:
		cond = andIssue(cond)
		cond = andComment(cond, reactionableID)
	default:
		panic(fmt.Errorf("unexpected kind %v", reactionable.GetKind()))
	}

	sess = sess.Where(cond)
	if page > 0 {
		sess = db.SetSessionPagination(sess, &db.ListOptions{Page: page, PageSize: pageSize})
	}
	reactions := make([]*issues_model.Reaction, 0, 10)
	if err := sess.Find(&reactions); err != nil {
		panic(fmt.Errorf("error while listing reactions: %v", err))
	}

	for _, reaction := range reactions {
		if _, err := reaction.LoadUser(ctx); err != nil {
			panic(fmt.Errorf("LoadUser(%+v): %w", reaction, err))
		}
	}

	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(reactions...)...)
}

func newReactions() f3_tree_generic.NodeDriverInterface {
	return &reactions{}
}
