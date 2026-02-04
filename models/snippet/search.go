// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package snippet

import (
	"context"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"

	"xorm.io/builder"
)

type SearchSnippetsOptions struct {
	db.ListOptions
	Actor     *user_model.User
	OwnerID   int64
	Keyword   string
	SortOrder string
}

// SearchSnippetsCondition returns Conditions for the given Options
func SearchSnippetsCondition(opts *SearchSnippetsOptions) builder.Cond {
	cond := builder.NewCond()

	if opts.Actor == nil {
		cond = cond.And(builder.Eq{"snippet.visibility": SnippetVisibilityPublic})
	} else if !opts.Actor.IsAdmin {
		ownCond := builder.NewCond()
		ownCond = ownCond.And(builder.Neq{"snippet.visibility": SnippetVisibilityPublic})
		ownCond = ownCond.And(builder.Eq{"snippet.owner_id": opts.Actor.ID})

		privateCond := builder.NewCond()
		privateCond = privateCond.Or(builder.Eq{"snippet.visibility": SnippetVisibilityPublic})
		privateCond = privateCond.Or(ownCond)

		cond = cond.And(privateCond)
	}

	if opts.OwnerID != 0 {
		cond = cond.And(builder.Eq{"snippet.owner_id": opts.OwnerID})
	}

	if opts.Keyword != "" {
		cond = cond.And(db.BuildCaseInsensitiveLike("snippet.name", opts.Keyword))
	}

	return cond
}

// SearchSnippets find Snippets by the given Options
func SearchSnippets(ctx context.Context, opts *SearchSnippetsOptions) (SnippetList, int64, error) {
	cond := SearchSnippetsCondition(opts)

	sess := db.GetEngine(ctx)

	count, err := sess.Where(cond).Count(new(Snippet))
	if err != nil {
		return nil, 0, err
	}

	if opts.SortOrder != "" {
		var orderBy string

		switch opts.SortOrder {
		case "newest":
			orderBy = "snippet.updated_unix DESC"
		case "oldest":
			orderBy = "snippet.updated_unix ASC"
		case "alphabetically":
			orderBy = "snippet.name ASC"
		case "reversealphabetically":
			orderBy = "snippet.name DESC"
		}

		if orderBy != "" {
			sess.OrderBy(orderBy)
		} else {
			sess.OrderBy("snippet.updated_unix DESC")
		}
	} else {
		sess.OrderBy("snippet.updated_unix DESC")
	}

	sess = sess.Where(cond)

	if opts.PageSize > 0 {
		sess = sess.Limit(opts.PageSize, (opts.Page-1)*opts.PageSize)
	}

	snippetList := make(SnippetList, 0)
	err = sess.Find(&snippetList)
	if err != nil {
		return nil, 0, err
	}

	return snippetList, count, nil
}

// CountSnippets return a number of all Snippets that match the Options
func CountSnippets(ctx context.Context, opts *SearchSnippetsOptions) (int64, error) {
	cond := SearchSnippetsCondition(opts)
	return db.GetEngine(ctx).Where(cond).Count(new(Snippet))
}
