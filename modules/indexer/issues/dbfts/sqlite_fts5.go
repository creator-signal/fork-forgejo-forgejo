// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

//go:build sqlite_fts5

package dbfts

import (
	"context"
	"strings"

	"forgejo.org/models/db"
	issue_model "forgejo.org/models/issues"
	indexer_internal "forgejo.org/modules/indexer/internal"
	inner_dbfts "forgejo.org/modules/indexer/internal/dbfts"
	indexer_db "forgejo.org/modules/indexer/issues/db"
	"forgejo.org/modules/indexer/issues/internal"

	"xorm.io/builder"
)

type SqliteIndexer struct {
	indexer_internal.Indexer
}

func createSqlite(ctx context.Context) error {
	_, err := db.GetEngine(ctx).Exec(
		"CREATE VIRTUAL TABLE IF NOT EXISTS issue_fts_idx_v1 USING fts5(title, body, comment, content='', contentless_delete=1)",
	)
	return err
}

func init() {
	registery["sqlite3"] = NewSqliteIndexer
}

func NewSqliteIndexer() internal.Indexer {
	return &SqliteIndexer{
		Indexer: inner_dbfts.NewIndexer(tableName, createSqlite),
	}
}

// Index the data to our virtual table
func (i *SqliteIndexer) Index(ctx context.Context, issues ...*internal.IndexerData) error {
	engine := db.GetEngine(ctx)
	sql := "INSERT OR REPLACE INTO issue_fts_idx_v1(rowid, title, body, comment) VALUES(?, ?, ?, ?)"
	for _, issue := range issues {
		if _, err := engine.Exec(sql, issue.ID, issue.Title, issue.Content, strings.Join(issue.Comments, " ")); err != nil {
			return err
		}
	}
	// _, err := engine.Exec("INSERT INTO issue_fts_idx_v1(issue_fts_idx_v1) VALUES('optimize')")
	return nil
}

// Delete the content from our virtual table
func (i *SqliteIndexer) Delete(ctx context.Context, issues ...int64) error {
	sql, args, err := builder.Delete(builder.In("rowid", issues)).From("issue_fts_idx_v1").ToSQL()
	if err != nil {
		return err
	}
	_, err = db.GetEngine(ctx).Exec(sql, args)
	return err
}

// Search searches for issues
func (i *SqliteIndexer) Search(ctx context.Context, options *internal.SearchOptions) (*internal.SearchResult, error) {
	cond := builder.NewCond()
	var priorityIssueIndex int64

	if tokens, err := options.Tokens(); err != nil {
		return nil, err
	} else if len(tokens) > 0 {
		var refs []int64
		match := builder.NewCond()
		for _, token := range tokens {
			match = match.Or(
				builder.Eq{tableName: token.Term},
				cond,
			)
			if ref, err := token.ParseIssueReference(); err == nil {
				refs = append(refs, ref)
				// cutting corners here
				priorityIssueIndex = ref
			}
		}
		switch len(refs) {
		case 0:
			break
		case 1:
			match = match.Or(
				builder.Eq{"`index`": refs[0]},
				cond,
			)
		default:
			match = match.Or(
				builder.In("`index`", refs),
				cond,
			)
		}
		cond = cond.And(match, cond)
	}

	opt, err := indexer_db.ToDBOptions(ctx, options)
	if err != nil {
		return nil, err
	}
	opt.PriorityIssueIndex = priorityIssueIndex

	if options.Paginator != nil && options.Paginator.PageSize == 0 {
		sess := db.GetEngine(ctx).
			Select("COUNT(issue.id) AS count").
			Table("issue").
			Join("INNER", "repository", "`issue`.repo_id = `repository`.id").
			Join("INNER", tableName, "`issue`.id = `issue_fts_idx_v1`.rowid")
		total, err := issue_model.CountIssuesFromSession(sess, opt, cond)
		if err != nil {
			return nil, err
		}
		return &internal.SearchResult{
			Total: total,
		}, nil
	}

	sess := db.GetEngine(ctx).
		Join("INNER", "repository", "`issue`.repo_id = `repository`.id").
		Join("INNER", "issue_fts_idx_v1", "`issue`.id = `issue_fts_idx_v1`.rowid")

	ids, total, err := issue_model.IssueIDsFromSession(sess, opt, cond)
	if err != nil {
		return nil, err
	}

	hits := make([]internal.Match, 0, len(ids))
	for _, id := range ids {
		hits = append(hits, internal.Match{ID: id})
	}
	return &internal.SearchResult{
		Total: total,
		Hits:  hits,
	}, nil
}
