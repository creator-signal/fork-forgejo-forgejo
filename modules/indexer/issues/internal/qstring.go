// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package internal

import (
	"context"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// labelFilter is a label name collected from the query plus whether it was
// quoted, which forces an exact match instead of the lenient one.
type labelFilter struct {
	name  string
	exact bool
}

// Parses the keyword and sets the
func (o *SearchOptions) WithKeyword(ctx context.Context, keyword string) (err error) {
	if keyword == "" {
		return nil
	}

	in := strings.NewReader(keyword)
	it := Tokenizer{in: in}

	var (
		tokens         []Token
		userNames      []string
		userFilter     []userFilter
		mustLabels     []labelFilter
		shouldLabels   []labelFilter
		excludedLabels []labelFilter
	)

	for token, err := it.next(); err == nil; token, err = it.next() {
		if token.Term == "" {
			continue
		}

		// Checked before the fuzzy-skip so that quoted names like
		// label:"good first issue" still parse as a filter. A quoted value
		// (Fuzzy == false) forces an exact match.
		if token.IsOf("label:") {
			lf := labelFilter{name: token.Term[len("label:"):], exact: !token.Fuzzy}
			switch token.Kind {
			case BoolOptNot:
				excludedLabels = append(excludedLabels, lf)
			case BoolOptMust:
				mustLabels = append(mustLabels, lf)
			default:
				shouldLabels = append(shouldLabels, lf)
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

	return o.resolveLabelNames(ctx, mustLabels, shouldLabels, excludedLabels)
}

// resolveLabelNames maps label filters collected from the query into the
// SearchOptions ID slices, following the convention used elsewhere in
// the query language: bare `label:foo` is SHOULD (OR among names),
// `+label:foo` is MUST (AND), `-label:foo` is NOT.
func (o *SearchOptions) resolveLabelNames(ctx context.Context, must, should, excluded []labelFilter) error {
	if len(must) == 0 && len(should) == 0 && len(excluded) == 0 {
		return nil
	}

	// A label name resolves to one ID only when the search is scoped to a
	// single repository and not also matching all public repositories.
	singleScope := !o.AllPublic && len(o.RepoIDs) == 1

	var lookup func(filters []labelFilter) ([]int64, error)
	if o.AllPublic {
		// Searching all public repositories: the label universe is unbounded,
		// so resolve names with an exact, indexer-friendly query rather than
		// loading every candidate label.
		lookup = func(filters []labelFilter) ([]int64, error) {
			names := make([]string, len(filters))
			for i, f := range filters {
				names[i] = f.name
			}
			return issues_model.GetLabelIDsByNames(ctx, names)
		}
	} else {
		labels, err := issues_model.GetLabelsByRepoIDs(ctx, o.RepoIDs)
		if err != nil {
			return err
		}
		exact := make(map[string][]int64)
		lenient := make(map[string][]int64)
		for _, l := range labels {
			exact[l.Name] = append(exact[l.Name], l.ID)
			key := normalizeLabelName(l.Name)
			lenient[key] = append(lenient[key], l.ID)
		}
		lookup = func(filters []labelFilter) ([]int64, error) {
			var ids []int64
			seen := make(map[int64]bool)
			add := func(id int64) {
				if !seen[id] {
					seen[id] = true
					ids = append(ids, id)
				}
			}
			for _, f := range filters {
				for _, id := range exact[f.name] {
					add(id)
				}
				// An unquoted name without a typed "/" also matches labels
				// that differ only by case, spaces or the "/" scope separator
				// (which the UI hides), so label:testpresent finds a scoped
				// "test/present". A typed "/" is kept literal and matches only
				// via the exact name above.
				if !f.exact && !strings.ContainsRune(f.name, '/') {
					for _, id := range lenient[normalizeLabelName(f.name)] {
						add(id)
					}
				}
			}
			return ids, nil
		}
	}

	// A name can resolve to several IDs (one per repo), and an issue cannot
	// have all of them, so MUST collapses into SHOULD unless the search is
	// scoped to a single repository. AND across name groups would need a new
	// indexer schema field.
	if !singleScope && len(must) > 0 {
		should = append(should, must...)
		must = nil
	}

	if len(must) > 0 {
		ids, err := lookup(must)
		if err != nil {
			return err
		}
		o.IncludedLabelIDs = append(o.IncludedLabelIDs, ids...)
	}
	if len(should) > 0 {
		ids, err := lookup(should)
		if err != nil {
			return err
		}
		// IncludedAnyLabelIDs is ignored by every backend when
		// IncludedLabelIDs is non-empty (whether set above or via the
		// URL `?labels=` parameter). Promote SHOULDs to MUST in that
		// case so the filter still applies — composing `?labels=X` or
		// `+label:X` with bare `label:Y` matches issues with both.
		if len(o.IncludedLabelIDs) > 0 {
			o.IncludedLabelIDs = append(o.IncludedLabelIDs, ids...)
		} else {
			o.IncludedAnyLabelIDs = append(o.IncludedAnyLabelIDs, ids...)
		}
	}
	if len(excluded) > 0 {
		ids, err := lookup(excluded)
		if err != nil {
			return err
		}
		o.ExcludedLabelIDs = append(o.ExcludedLabelIDs, ids...)
	}
	return nil
}

// normalizeLabelName is the key used for lenient label matching: lowercased,
// with whitespace and the "/" scope separator stripped (for exclusive labels
// the slash doesn't appear in the UI, so label:testpresent should still match
// a scoped "test/present").
func normalizeLabelName(name string) string {
	return strings.Map(func(r rune) rune {
		if r == '/' || unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, name)
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
