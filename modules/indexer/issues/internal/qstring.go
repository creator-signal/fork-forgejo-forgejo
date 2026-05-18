// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
)

type BoolOpt int

const (
	BoolOptMust BoolOpt = iota
	BoolOptShould
	BoolOptNot
)

type Token struct {
	Term  string
	Kind  BoolOpt
	Fuzzy bool
}

// Helper function to check if the term starts with a prefix.
func (tk *Token) IsOf(prefix string) bool {
	return strings.HasPrefix(tk.Term, prefix) && len(tk.Term) > len(prefix)
}

func (tk *Token) ParseIssueReference() (int64, error) {
	term := tk.Term
	if len(term) > 1 && (term[0] == '#' || term[0] == '!') {
		term = term[1:]
	}
	return strconv.ParseInt(term, 10, 64)
}

type Tokenizer struct {
	in *strings.Reader
}

func (t *Tokenizer) next() (tk Token, err error) {
	var (
		sb strings.Builder
		r  rune
	)
	tk.Kind = BoolOptShould
	tk.Fuzzy = true

	// skip all leading white space
	for {
		if r, _, err = t.in.ReadRune(); err != nil || r != ' ' {
			break
		}
	}
	if err != nil {
		return tk, err
	}

	// check for +/- op, increment to the next rune in both cases
	switch r {
	case '+':
		tk.Kind = BoolOptMust
		r, _, err = t.in.ReadRune()
	case '-':
		tk.Kind = BoolOptNot
		r, _, err = t.in.ReadRune()
	}
	if err != nil {
		return tk, err
	}

	// parse the string, escaping special characters
	for esc := false; err == nil; r, _, err = t.in.ReadRune() {
		if esc {
			if !strings.ContainsRune("+-\\\"", r) {
				sb.WriteRune('\\')
			}
			sb.WriteRune(r)
			esc = false
			continue
		}
		switch r {
		case '\\':
			esc = true
		case '"':
			if !tk.Fuzzy {
				goto nextEnd
			}
			tk.Fuzzy = false
		case ' ', '\t':
			if tk.Fuzzy {
				goto nextEnd
			}
			sb.WriteRune(r)
		default:
			sb.WriteRune(r)
		}
	}
nextEnd:

	tk.Term = sb.String()
	if err == io.EOF {
		err = nil
	} // do not consider EOF as an error at the end
	return tk, err
}

type userFilter int

const (
	userFilterAuthor userFilter = iota
	userFilterAssign
	userFilterMention
	userFilterReview
)

// Parses the keyword and sets the
func (o *SearchOptions) WithKeyword(ctx context.Context, keyword string) (err error) {
	if keyword == "" {
		return nil
	}

	in := strings.NewReader(keyword)
	it := Tokenizer{in: in}

	var (
		tokens             []Token
		userNames          []string
		userFilter         []userFilter
		includedLabelNames []string
		excludedLabelNames []string
	)

	for token, err := it.next(); err == nil; token, err = it.next() {
		if token.Term == "" {
			continue
		}

		// Checked before the fuzzy-skip so that quoted names like
		// label:"good first issue" still parse as a filter.
		if token.IsOf("label:") {
			name := token.Term[len("label:"):]
			if token.Kind == BoolOptNot {
				excludedLabelNames = append(excludedLabelNames, name)
			} else {
				includedLabelNames = append(includedLabelNames, name)
			}
			continue
		}

		// For an exact search (wrapped in quotes)
		// push the token to the list.
		if !token.Fuzzy {
			tokens = append(tokens, token)
			continue
		}

		// Otherwise, try to match the token with a preset filter.
		switch {
		// is:open  => open & -is:open => closed
		case token.Term == "is:open":
			o.IsClosed = optional.Some(token.Kind == BoolOptNot)

		// Similarly, is:closed & -is:closed
		case token.Term == "is:closed":
			o.IsClosed = optional.Some(token.Kind != BoolOptNot)

		// Mirrors ?labels=0. Do not consider -no:label.
		case token.Term == "no:label":
			o.NoLabelOnly = true

		// The rest of the presets MUST NOT be a negation.
		case token.Kind == BoolOptNot:
			tokens = append(tokens, token)

		// is:all: Do not consider -is:all.
		case token.Term == "is:all":
			o.IsClosed = optional.None[bool]()

		// sort:<by>:[ asc | desc ],
		case token.IsOf("sort:"):
			o.SortBy = parseSortBy(token.Term[5:])

		// modified:[ < | > ]<date>.
		// for example, modified:>2025-08-29
		case token.IsOf("modified:"):
			switch token.Term[9] {
			case '>':
				o.UpdatedAfterUnix = toUnix(token.Term[10:])
			case '<':
				o.UpdatedBeforeUnix = toUnix(token.Term[10:])
			default:
				t := toUnix(token.Term[9:])
				o.UpdatedAfterUnix = t
				o.UpdatedBeforeUnix = t
			}

		// for user filter's
		// append the names and roles
		case token.IsOf("author:"):
			userNames = append(userNames, token.Term[7:])
			userFilter = append(userFilter, userFilterAuthor)
		case token.IsOf("assignee:"):
			userNames = append(userNames, token.Term[9:])
			userFilter = append(userFilter, userFilterAssign)
		case token.IsOf("review:"):
			userNames = append(userNames, token.Term[7:])
			userFilter = append(userFilter, userFilterReview)
		case token.IsOf("mentions:"):
			userNames = append(userNames, token.Term[9:])
			userFilter = append(userFilter, userFilterMention)

		default:
			tokens = append(tokens, token)
		}
	}
	if err != nil && err != io.EOF {
		return err
	}

	o.Tokens = tokens

	ids, err := user.GetUserIDsByNames(ctx, userNames, true)
	if err != nil {
		return err
	}

	for i, id := range ids {
		// Skip all invalid IDs.
		// Hopefully this won't be too astonishing for the user.
		if id <= 0 {
			continue
		}
		val := optional.Some(id)
		switch userFilter[i] {
		case userFilterAuthor:
			o.PosterID = val
		case userFilterAssign:
			o.AssigneeID = val
		case userFilterReview:
			o.ReviewedID = val
		case userFilterMention:
			o.MentionID = val
		}
	}

	return o.resolveLabelNames(ctx, includedLabelNames, excludedLabelNames)
}

// resolveLabelNames appends label IDs from `label:` / `-label:` tokens
// to o, so the query composes with any IDs already set from `?labels=`.
func (o *SearchOptions) resolveLabelNames(ctx context.Context, included, excluded []string) error {
	if len(included) == 0 && len(excluded) == 0 {
		return nil
	}

	// Single repo: names map 1:1 to IDs, so the per-repo lookup keeps
	// AND semantics via IncludedLabelIDs.
	if len(o.RepoIDs) == 1 {
		if len(included) > 0 {
			ids, err := issues_model.GetLabelIDsInRepoByNames(ctx, o.RepoIDs[0], included)
			if err != nil {
				return err
			}
			o.IncludedLabelIDs = append(o.IncludedLabelIDs, ids...)
		}
		if len(excluded) > 0 {
			ids, err := issues_model.GetLabelIDsInRepoByNames(ctx, o.RepoIDs[0], excluded)
			if err != nil {
				return err
			}
			o.ExcludedLabelIDs = append(o.ExcludedLabelIDs, ids...)
		}
		return nil
	}

	// Cross-repo: a name can resolve to many IDs (one per repo), so
	// use IncludedAnyLabelIDs (at-least-one). Two or more positive
	// names degrade to a union; a proper AND across name groups would
	// need a new indexer schema field.
	if len(included) > 0 {
		ids, err := issues_model.GetLabelIDsByNames(ctx, included)
		if err != nil {
			return err
		}
		if len(o.IncludedLabelIDs) == 0 {
			o.IncludedAnyLabelIDs = append(o.IncludedAnyLabelIDs, ids...)
		}
	}
	if len(excluded) > 0 {
		ids, err := issues_model.GetLabelIDsByNames(ctx, excluded)
		if err != nil {
			return err
		}
		o.ExcludedLabelIDs = append(o.ExcludedLabelIDs, ids...)
	}
	return nil
}

func toUnix(value string) optional.Option[int64] {
	time, err := time.Parse(time.DateOnly, value)
	if err != nil {
		log.Warn("Failed to parse date '%v'", err)
		return optional.None[int64]()
	}

	return optional.Some(time.Unix())
}

func parseSortBy(sortBy string) SortBy {
	switch sortBy {
	case "created:asc":
		return SortByCreatedAsc
	case "created:desc":
		return SortByCreatedDesc
	case "comments:asc":
		return SortByCommentsAsc
	case "comments:desc":
		return SortByCommentsDesc
	case "updated:asc":
		return SortByUpdatedAsc
	case "updated:desc":
		return SortByUpdatedDesc
	case "deadline:asc":
		return SortByDeadlineAsc
	case "deadline:desc":
		return SortByDeadlineDesc
	default:
		return SortByScore
	}
}
