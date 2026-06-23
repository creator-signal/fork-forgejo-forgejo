// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStarRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	const userID = 2
	const repoID = 1
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.StarRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.StarRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.StarRepo(db.DefaultContext, userID, repoID, false))
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
}

func TestIsStaring(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	assert.True(t, repo_model.IsStaring(db.DefaultContext, 2, 4))
	assert.False(t, repo_model.IsStaring(db.DefaultContext, 3, 4))
}

func TestGetVisibleStarCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	profileUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	t.Logf("profileUser: ID=%d Name=%s", profileUser.ID, profileUser.Name)

	count, err := repo_model.GetVisibleStarCount(t.Context(), profileUser, profileUser, "")
	t.Logf("owner view: count=%d err=%v", count, err)
	assert.NoError(t, err)
	assert.Equal(t, 5, count, "owner should see all their own stars")

	count, err = repo_model.GetVisibleStarCount(t.Context(), profileUser, nil, "")
	t.Logf("anonymous view: count=%d err=%v", count, err)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "anon should only see public repos with public owners")

	user5 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5})
	t.Logf("user5: ID=%d Name=%s", user5.ID, user5.Name)

	count, err = repo_model.GetVisibleStarCount(t.Context(), profileUser, user5, "")
	t.Logf("user5 view: count=%d err=%v", count, err)
	assert.NoError(t, err)
	assert.Equal(t, 2, count, "unrelated signed-in user should only see public repos")

	user15 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 15})
	t.Logf("user15: ID=%d Name=%s", user15.ID, user15.Name)

	count, err = repo_model.GetVisibleStarCount(t.Context(), profileUser, user15, "")
	t.Logf("user15 (collaborator) view: count=%d err=%v", count, err)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "collaborator should see public repos + their collab private repo")

	user20 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 20})
	t.Logf("user20: ID=%d Name=%s", user20.ID, user20.Name)

	count, err = repo_model.GetVisibleStarCount(t.Context(), profileUser, user20, "")
	t.Logf("user20 (org member) view: count=%d err=%v", count, err)
	assert.NoError(t, err)
	assert.Equal(t, 3, count, "org member should see public repos + org's private repo")
}
func TestRepository_GetStargazers(t *testing.T) {
	// repo with stargazers
	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
	gazers, err := repo_model.GetStargazers(db.DefaultContext, repo, db.ListOptions{Page: 0})
	require.NoError(t, err)
	if assert.Len(t, gazers, 1) {
		assert.Equal(t, int64(2), gazers[0].ID)
	}
}

func TestRepository_GetStargazers2(t *testing.T) {
	// repo with stargazers
	require.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
	gazers, err := repo_model.GetStargazers(db.DefaultContext, repo, db.ListOptions{Page: 0})
	require.NoError(t, err)
	assert.Empty(t, gazers)
}

func TestClearRepoStars(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	const userID = 2
	const repoID = 1
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.StarRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.StarRepo(db.DefaultContext, userID, repoID, false))
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID, RepoID: repoID})
	require.NoError(t, repo_model.ClearRepoStars(db.DefaultContext, repoID))
	unittest.AssertNotExistsBean(t, &repo_model.Star{UID: userID, RepoID: repoID})

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	gazers, err := repo_model.GetStargazers(db.DefaultContext, repo, db.ListOptions{Page: 0})
	require.NoError(t, err)
	assert.Empty(t, gazers)
}
