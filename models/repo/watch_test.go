// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsWatching(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	assert.True(t, repo_model.IsWatching(db.DefaultContext, 1, 1))
	assert.True(t, repo_model.IsWatching(db.DefaultContext, 4, 1))
	assert.True(t, repo_model.IsWatching(db.DefaultContext, 11, 1))

	assert.False(t, repo_model.IsWatching(db.DefaultContext, 1, 5))
	assert.False(t, repo_model.IsWatching(db.DefaultContext, 8, 1))
	assert.False(t, repo_model.IsWatching(db.DefaultContext, unittest.NonexistentID, unittest.NonexistentID))
}

func TestGetWatchers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	watches, err := repo_model.GetWatchers(db.DefaultContext, repo.ID)
	require.NoError(t, err)
	// One watchers are inactive, thus minus 1
	assert.Len(t, watches, repo.NumWatches-1)
	for _, watch := range watches {
		assert.Equal(t, repo.ID, watch.RepoID)
	}

	watches, err = repo_model.GetWatchers(db.DefaultContext, unittest.NonexistentID)
	require.NoError(t, err)
	assert.Empty(t, watches)
}

func TestRepository_GetWatchers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	watchers, err := repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, repo.NumWatches)
	for _, watcher := range watchers {
		unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: watcher.ID, RepoID: repo.ID})
	}

	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 9})
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Empty(t, watchers)
}

func TestWatchIfAuto(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	watchers, err := repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, repo.NumWatches)

	setting.Service.AutoWatchOnChanges = false

	prevCount := repo.NumWatches

	// Must not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 8, 1))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 10, 1))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	setting.Service.AutoWatchOnChanges = true

	// Must not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 8, 1))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should not add watch
	// We simply don't WatchIfAuto
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 12, 1))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount+1)

	// Should remove watch, inhibit from adding auto
	require.NoError(t, repo_model.WatchRepoExplicitly(db.DefaultContext, 12, 1, repo_model.WatchNoneSelection))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Must not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 12, 1))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)
}

func TestUnwatchRepos(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: 4, RepoID: 1})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: 4, RepoID: 2})

	err := repo_model.UnwatchRepos(db.DefaultContext, 4, []int64{1, 2})
	require.NoError(t, err)

	unittest.AssertCount(t, &repo_model.Watch{UserID: 4, RepoID: 1, Source: repo_model.WatchSourceExplicit, WatchSelectionIssues: false, WatchSelectionPullRequests: false, WatchSelectionReleases: false}, 1)
	unittest.AssertCount(t, &repo_model.Watch{UserID: 4, RepoID: 2, Source: repo_model.WatchSourceExplicit, WatchSelectionIssues: false, WatchSelectionPullRequests: false, WatchSelectionReleases: false}, 1)
}
