// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"context"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	"forgejo.org/models/user"
	"forgejo.org/modules/optional"

	_ "forgejo.org/modules/testimport"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testIssueQueryStringOpt struct {
	Keyword string
	Results []Token
}

var testOpts = []testIssueQueryStringOpt{
	{
		Keyword: "Hello",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "Hello World",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "Hello  World",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: " Hello World ",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "+Hello +World",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptMust,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptMust,
			},
		},
	},
	{
		Keyword: "+Hello World",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptMust,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "+Hello -World",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptMust,
			},
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptNot,
			},
		},
	},
	{
		Keyword: "\"Hello World\"",
		Results: []Token{
			{
				Term:  "Hello World",
				Fuzzy: false,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "+\"Hello World\"",
		Results: []Token{
			{
				Term:  "Hello World",
				Fuzzy: false,
				Kind:  BoolOptMust,
			},
		},
	},
	{
		Keyword: "-\"Hello World\"",
		Results: []Token{
			{
				Term:  "Hello World",
				Fuzzy: false,
				Kind:  BoolOptNot,
			},
		},
	},
	{
		Keyword: "\"+Hello -World\"",
		Results: []Token{
			{
				Term:  "+Hello -World",
				Fuzzy: false,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\\+Hello", // \+Hello => +Hello
		Results: []Token{
			{
				Term:  "+Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\\\\Hello", // \\Hello => \Hello
		Results: []Token{
			{
				Term:  "\\Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\\\"Hello", // \"Hello => "Hello
		Results: []Token{
			{
				Term:  "\"Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\\",
		Results: nil,
	},
	{
		Keyword: "\"",
		Results: nil,
	},
	{
		Keyword: "Hello \\",
		Results: []Token{
			{
				Term:  "Hello",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\"\"",
		Results: nil,
	},
	{
		Keyword: "\" World \"",
		Results: []Token{
			{
				Term:  " World ",
				Fuzzy: false,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "\"\" World \"\"",
		Results: []Token{
			{
				Term:  "World",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
	{
		Keyword: "Best \"Hello World\" Ever",
		Results: []Token{
			{
				Term:  "Best",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
			{
				Term:  "Hello World",
				Fuzzy: false,
				Kind:  BoolOptShould,
			},
			{
				Term:  "Ever",
				Fuzzy: true,
				Kind:  BoolOptShould,
			},
		},
	},
}

func TestIssueQueryString(t *testing.T) {
	var opt SearchOptions
	ctx := t.Context()
	for _, res := range testOpts {
		t.Run(res.Keyword, func(t *testing.T) {
			require.NoError(t, opt.WithKeyword(ctx, res.Keyword))
			assert.Equal(t, res.Results, opt.Tokens)
		})
	}
}

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func TestIssueQueryStringWithFilters(t *testing.T) {
	// we don't need all the fixures
	// insert only one single test user
	require.NoError(t, user.CreateUser(t.Context(), &user.User{
		ID:        2,
		Name:      "test",
		LowerName: "test",
		Email:     "test@localhost",
	}))

	for _, c := range []struct {
		Keyword string
		Opts    *SearchOptions
	}{
		// Generic Cases
		{
			Keyword: "modified:>2025-08-28",
			Opts: &SearchOptions{
				UpdatedAfterUnix: optional.Some(int64(1756339200)),
			},
		},
		{
			Keyword: "modified:<2025-08-28",
			Opts: &SearchOptions{
				UpdatedBeforeUnix: optional.Some(int64(1756339200)),
			},
		},
		{
			Keyword: "modified:>2025-08-28 modified:<2025-08-28",
			Opts: &SearchOptions{
				UpdatedAfterUnix:  optional.Some(int64(1756339200)),
				UpdatedBeforeUnix: optional.Some(int64(1756339200)),
			},
		},
		{
			Keyword: "modified:2025-08-28",
			Opts: &SearchOptions{
				UpdatedAfterUnix:  optional.Some(int64(1756339200)),
				UpdatedBeforeUnix: optional.Some(int64(1756339200)),
			},
		},
		{
			Keyword: "assignee:test",
			Opts: &SearchOptions{
				AssigneeID: optional.Some(int64(2)),
			},
		},
		{
			Keyword: "assignee:test hi",
			Opts: &SearchOptions{
				AssigneeID: optional.Some(int64(2)),
				Tokens: []Token{
					{
						Term:  "hi",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
				},
			},
		},
		{
			Keyword: "mentions:test",
			Opts: &SearchOptions{
				MentionID: optional.Some(int64(2)),
			},
		},
		{
			Keyword: "review:test",
			Opts: &SearchOptions{
				ReviewedID: optional.Some(int64(2)),
			},
		},
		{
			Keyword: "author:test",
			Opts: &SearchOptions{
				PosterID: optional.Some(int64(2)),
			},
		},
		{
			Keyword: "sort:updated:asc",
			Opts: &SearchOptions{
				SortBy: SortByUpdatedAsc,
			},
		},
		{
			Keyword: "sort:test",
			Opts: &SearchOptions{
				SortBy: SortByScore,
			},
		},
		{
			Keyword: "test author:test mentions:test modified:<2025-08-28 sort:comments:desc",
			Opts: &SearchOptions{
				Tokens: []Token{
					{
						Term:  "test",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
				},
				MentionID:         optional.Some(int64(2)),
				PosterID:          optional.Some(int64(2)),
				UpdatedBeforeUnix: optional.Some(int64(1756339200)),
				SortBy:            SortByCommentsDesc,
			},
		},

		// Edge Cases
		{
			Keyword: "author:",
			Opts: &SearchOptions{
				Tokens: []Token{
					{
						Term:  "author:",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
				},
			},
		},
		{
			Keyword: "author:testt",
			Opts:    &SearchOptions{},
		},
		{
			Keyword: "author: test",
			Opts: &SearchOptions{
				Tokens: []Token{
					{
						Term:  "author:",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
					{
						Term:  "test",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
				},
			},
		},
		{
			Keyword: "modified:",
			Opts: &SearchOptions{
				Tokens: []Token{
					{
						Term:  "modified:",
						Kind:  BoolOptShould,
						Fuzzy: true,
					},
				},
			},
		},
	} {
		t.Run(c.Keyword, func(t *testing.T) {
			opts := &SearchOptions{}
			require.NoError(t, opts.WithKeyword(context.Background(), c.Keyword))
			assert.Equal(t, c.Opts, opts)
		})
	}
}

func TestIssueQueryStringWithLabelFilters(t *testing.T) {
	// Two repos: one for the repo-scoped path, both for the cross-repo path.
	for _, l := range []*issues_model.Label{
		{ID: 200, RepoID: 100, Name: "bug", Color: "#ff0000"},
		{ID: 201, RepoID: 100, Name: "critical", Color: "#ffaa00"},
		{ID: 202, RepoID: 100, Name: "wontfix", Color: "#888888"},
		{ID: 203, RepoID: 100, Name: "good first issue", Color: "#00ff00"},
		{ID: 210, RepoID: 101, Name: "bug", Color: "#ff0000"},
	} {
		require.NoError(t, issues_model.NewLabel(t.Context(), l))
	}

	for _, c := range []struct {
		Name    string
		Initial *SearchOptions
		Keyword string
		Want    *SearchOptions
	}{
		{
			Name:    "single positive label",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "label:bug",
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				IncludedLabelIDs: []int64{200},
			},
		},
		{
			Name:    "single negative label",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "-label:wontfix",
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				ExcludedLabelIDs: []int64{202},
			},
		},
		{
			// GetLabelIDsInRepoByNames orders by name asc, so "bug"
			// (200) comes before "critical" (201).
			Name:    "two positive labels are AND-ed",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "label:bug label:critical",
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				IncludedLabelIDs: []int64{200, 201},
			},
		},
		{
			Name:    "label name with spaces requires quoting",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: `label:"good first issue"`,
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				IncludedLabelIDs: []int64{203},
			},
		},
		{
			Name:    "no:label sets NoLabelOnly",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "no:label",
			Want: &SearchOptions{
				RepoIDs:     []int64{100},
				NoLabelOnly: true,
			},
		},
		{
			Name:    "unknown label name is silently dropped",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "label:nonexistent",
			Want:    &SearchOptions{RepoIDs: []int64{100}},
		},
		{
			Name:    "appends to existing IncludedLabelIDs from URL ?labels=",
			Initial: &SearchOptions{RepoIDs: []int64{100}, IncludedLabelIDs: []int64{42}},
			Keyword: "label:bug",
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				IncludedLabelIDs: []int64{42, 200},
			},
		},
		{
			Name:    "mixing positive and negative labels in one query",
			Initial: &SearchOptions{RepoIDs: []int64{100}},
			Keyword: "label:bug -label:wontfix",
			Want: &SearchOptions{
				RepoIDs:          []int64{100},
				IncludedLabelIDs: []int64{200},
				ExcludedLabelIDs: []int64{202},
			},
		},
	} {
		t.Run(c.Name, func(t *testing.T) {
			require.NoError(t, c.Initial.WithKeyword(t.Context(), c.Keyword))
			assert.Equal(t, c.Want, c.Initial)
		})
	}

	// Cross-repo path: GetLabelIDsByNames is unordered, so check the
	// label slice with ElementsMatch and the rest field by field.
	t.Run("cross-repo search uses IncludedAnyLabelIDs", func(t *testing.T) {
		opts := &SearchOptions{RepoIDs: []int64{100, 101}}
		require.NoError(t, opts.WithKeyword(t.Context(), "label:bug"))
		assert.Equal(t, []int64{100, 101}, opts.RepoIDs)
		assert.ElementsMatch(t, []int64{200, 210}, opts.IncludedAnyLabelIDs)
		assert.Empty(t, opts.IncludedLabelIDs)
		assert.Empty(t, opts.ExcludedLabelIDs)
	})
}

func TestToken_ParseIssueReference(t *testing.T) {
	var tk Token
	{
		tk.Term = "123"
		id, err := tk.ParseIssueReference()
		require.NoError(t, err)
		assert.Equal(t, int64(123), id)
	}
	{
		tk.Term = "#123"
		id, err := tk.ParseIssueReference()
		require.NoError(t, err)
		assert.Equal(t, int64(123), id)
	}
	{
		tk.Term = "!123"
		id, err := tk.ParseIssueReference()
		require.NoError(t, err)
		assert.Equal(t, int64(123), id)
	}
	{
		tk.Term = "text"
		_, err := tk.ParseIssueReference()
		require.Error(t, err)
	}
	{
		tk.Term = ""
		_, err := tk.ParseIssueReference()
		require.Error(t, err)
	}
}
