// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package setting

import (
	"sort"
	"testing"

	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// org3 owns repo3, repo5 and repo21 in the fixtures and user2 is in its owners
// team; org25 owns no repository and user1 is an instance admin.
const (
	orgWithRepos    = int64(3)
	orgWithoutRepos = int64(25)
	ownerUserID     = int64(2)
	adminUserID     = int64(1)
)

// runRepos executes the handler for the given query string and returns the
// resulting context so the caller can assert on ctx.Data.
func runRepos(t *testing.T, query string, orgID, doerID int64) *context.Context {
	t.Helper()
	ctx, _ := contexttest.MockContext(t, "/org/org/settings/repos"+query)
	ctx.Doer = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: doerID})
	ctx.Org = &context.Organization{
		Organization: unittest.AssertExistsAndLoadBean(t, &organization.Organization{ID: orgID}),
	}
	Repos(ctx)
	return ctx
}

func repoNames(repos repo_model.RepositoryList) []string {
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.LowerName
	}
	return names
}

// The default sort is "recentupdate", and any unknown value must fall back to
// it rather than being passed through to the query layer.
func TestReposSortType(t *testing.T) {
	unittest.PrepareTestEnv(t)

	cases := []struct{ query, expected string }{
		{"", "recentupdate"},
		{"?sort=not_a_valid_sort", "recentupdate"},
		{"?sort=alphabetically", "alphabetically"},
		{"?sort=gitsize", "gitsize"},
		{"?sort=lfssize", "lfssize"},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			ctx := runRepos(t, tc.query, orgWithRepos, ownerUserID)
			assert.Equal(t, tc.expected, ctx.Data["SortType"])
		})
	}
}

// Sorting must actually order the returned repositories
//
//	Sizes are all 0 in the fixtures, so alphabetical ordering is the deterministic case we can assert on.
func TestReposSortOrdersResults(t *testing.T) {
	unittest.PrepareTestEnv(t)

	asc := repoNames(runRepos(t, "?sort=alphabetically", orgWithRepos, ownerUserID).Data["Repos"].(repo_model.RepositoryList))
	desc := repoNames(runRepos(t, "?sort=reversealphabetically", orgWithRepos, ownerUserID).Data["Repos"].(repo_model.RepositoryList))

	require.NotEmpty(t, asc)
	assert.True(t, sort.SliceIsSorted(asc, func(i, j int) bool { return asc[i] < asc[j] }),
		"ascending sort must order repositories by name: %v", asc)
	assert.True(t, sort.SliceIsSorted(desc, func(i, j int) bool { return desc[i] > desc[j] }),
		"descending sort must reverse the order: %v", desc)
	assert.NotEqual(t, asc, desc, "the two sort directions must differ")
}

// Only repositories owned by the organization must be listed, private ones
// included.
func TestReposOnlyOrgRepos(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := runRepos(t, "", orgWithRepos, ownerUserID)
	repos := ctx.Data["Repos"].(repo_model.RepositoryList)
	require.NotEmpty(t, repos)
	hasPrivate := false
	for _, r := range repos {
		assert.Equal(t, orgWithRepos, r.OwnerID)
		hasPrivate = hasPrivate || r.IsPrivate
	}
	assert.True(t, hasPrivate, "the owner must see the organization private repositories")
}

// Total must reflect the full repository count across all pages, and the pages
// must be disjoint; page indexes <= 0 must clamp to the first page.
func TestReposPagination(t *testing.T) {
	unittest.PrepareTestEnv(t)
	defer test.MockVariableValue(&setting.UI.Admin.RepoPagingNum, 2)()

	ctx := runRepos(t, "?sort=alphabetically&page=1", orgWithRepos, ownerUserID)
	page1 := ctx.Data["Repos"].(repo_model.RepositoryList)
	total := ctx.Data["Total"].(int64)
	require.Len(t, page1, 2, "the page must be capped at the configured page size")
	assert.Greater(t, total, int64(2), "Total must count every repository, not just the current page")

	page2 := runRepos(t, "?sort=alphabetically&page=2", orgWithRepos, ownerUserID).Data["Repos"].(repo_model.RepositoryList)
	require.NotEmpty(t, page2)
	ids := make(map[int64]bool, len(page1))
	for _, r := range page1 {
		ids[r.ID] = true
	}
	for _, r := range page2 {
		assert.False(t, ids[r.ID], "page 2 must not repeat repositories from page 1 (repo %d)", r.ID)
	}

	// page=0 and page=-1 must behave like page 1.
	for _, q := range []string{"?sort=alphabetically&page=0", "?sort=alphabetically&page=-1"} {
		clamped := runRepos(t, q, orgWithRepos, ownerUserID).Data["Repos"].(repo_model.RepositoryList)
		assert.Equal(t, repoNames(page1), repoNames(clamped), "%s must clamp to page 1", q)
	}
}

// An organization that owns no repositories must still render successfully with
// an empty list (Total == 0), exercising the {{else}} branch of the template.
func TestReposNoRepos(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx := runRepos(t, "", orgWithoutRepos, adminUserID)
	assert.Empty(t, ctx.Data["Repos"].(repo_model.RepositoryList))
	assert.Equal(t, int64(0), ctx.Data["Total"].(int64))
}
