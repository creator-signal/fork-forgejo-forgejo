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
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 8, 1, true))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 10, 1, true))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	setting.Service.AutoWatchOnChanges = true

	// Must not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 8, 1, true))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 12, 1, false))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Should add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 12, 1, true))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount+1)

	// Should remove watch, inhibit from adding auto
	require.NoError(t, repo_model.WatchRepo(db.DefaultContext, 12, 1, false))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)

	// Must not add watch
	require.NoError(t, repo_model.WatchIfAuto(db.DefaultContext, 12, 1, true))
	watchers, err = repo_model.GetRepoWatchers(db.DefaultContext, repo.ID, db.ListOptions{Page: 1})
	require.NoError(t, err)
	assert.Len(t, watchers, prevCount)
}

func TestWatchRepoMode(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 0)

	require.NoError(t, repo_model.WatchRepoMode(db.DefaultContext, 12, 1, repo_model.WatchModeAuto))
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 1)
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1, Mode: repo_model.WatchModeAuto}, 1)

	require.NoError(t, repo_model.WatchRepoMode(db.DefaultContext, 12, 1, repo_model.WatchModeNormal))
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 1)
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1, Mode: repo_model.WatchModeNormal}, 1)

	require.NoError(t, repo_model.WatchRepoMode(db.DefaultContext, 12, 1, repo_model.WatchModeDont))
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 1)
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1, Mode: repo_model.WatchModeDont}, 1)

	require.NoError(t, repo_model.WatchRepoMode(db.DefaultContext, 12, 1, repo_model.WatchModeNone))
	unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 0)
}

func TestUnwatchRepos(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: 4, RepoID: 1})
	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{UserID: 4, RepoID: 2})

	err := repo_model.UnwatchRepos(db.DefaultContext, 4, []int64{1, 2})
	require.NoError(t, err)

	unittest.AssertNotExistsBean(t, &repo_model.Watch{UserID: 4, RepoID: 1})
	unittest.AssertNotExistsBean(t, &repo_model.Watch{UserID: 4, RepoID: 2})
}

func TestWatchEventType(t *testing.T) {
	t.Run("Bitmask values", func(t *testing.T) {
		assert.EqualValues(t, 1, repo_model.WatchEventIssues)
		assert.EqualValues(t, 2, repo_model.WatchEventPullRequests)
		assert.EqualValues(t, 4, repo_model.WatchEventReleases)
		assert.EqualValues(t, 7, repo_model.WatchEventAll)
	})

	t.Run("WatchesIssues", func(t *testing.T) {
		assert.True(t, repo_model.WatchEventIssues.WatchesIssues())
		assert.True(t, repo_model.WatchEventAll.WatchesIssues())
		assert.True(t, repo_model.WatchEventType(3).WatchesIssues()) // Issues + PRs
		assert.False(t, repo_model.WatchEventPullRequests.WatchesIssues())
		assert.False(t, repo_model.WatchEventReleases.WatchesIssues())
	})

	t.Run("WatchesPullRequests", func(t *testing.T) {
		assert.True(t, repo_model.WatchEventPullRequests.WatchesPullRequests())
		assert.True(t, repo_model.WatchEventAll.WatchesPullRequests())
		assert.True(t, repo_model.WatchEventType(3).WatchesPullRequests()) // Issues + PRs
		assert.False(t, repo_model.WatchEventIssues.WatchesPullRequests())
		assert.False(t, repo_model.WatchEventReleases.WatchesPullRequests())
	})

	t.Run("WatchesReleases", func(t *testing.T) {
		assert.True(t, repo_model.WatchEventReleases.WatchesReleases())
		assert.True(t, repo_model.WatchEventAll.WatchesReleases())
		assert.True(t, repo_model.WatchEventType(5).WatchesReleases()) // Issues + Releases
		assert.False(t, repo_model.WatchEventIssues.WatchesReleases())
		assert.False(t, repo_model.WatchEventPullRequests.WatchesReleases())
	})
}

func TestWatchGetWatchEvents(t *testing.T) {
	t.Run("Returns stored events", func(t *testing.T) {
		watch := &repo_model.Watch{WatchEvents: repo_model.WatchEventIssues}
		assert.Equal(t, repo_model.WatchEventIssues, watch.GetWatchEvents())

		watch = &repo_model.Watch{WatchEvents: repo_model.WatchEventType(3)}
		assert.Equal(t, repo_model.WatchEventType(3), watch.GetWatchEvents())
	})

	t.Run("Returns all events when zero", func(t *testing.T) {
		watch := &repo_model.Watch{WatchEvents: 0}
		assert.Equal(t, repo_model.WatchEventAll, watch.GetWatchEvents())
	})
}

func TestGetDefaultWatchEvents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Falls back to instance setting", func(t *testing.T) {
		setting.Service.DefaultWatchEvents = 3 // Issues + PRs
		events := repo_model.GetDefaultWatchEvents(db.DefaultContext, 1)
		assert.Equal(t, repo_model.WatchEventType(3), events)
	})

	t.Run("Falls back to all events when instance setting is zero", func(t *testing.T) {
		setting.Service.DefaultWatchEvents = 0
		events := repo_model.GetDefaultWatchEvents(db.DefaultContext, 1)
		assert.Equal(t, repo_model.WatchEventAll, events)
	})
}

func TestWatchRepoWithEvents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Creates watch with specific events", func(t *testing.T) {
		// User 12 is not watching repo 1
		unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 1}, 0)

		// Watch with only issues
		err := repo_model.WatchRepoWithEvents(db.DefaultContext, 12, 1, repo_model.WatchEventIssues)
		require.NoError(t, err)

		watch, err := repo_model.GetWatch(db.DefaultContext, 12, 1)
		require.NoError(t, err)
		assert.Equal(t, repo_model.WatchEventIssues, watch.WatchEvents)
		assert.True(t, repo_model.IsWatchMode(watch.Mode))

		// Clean up
		require.NoError(t, repo_model.WatchRepo(db.DefaultContext, 12, 1, false))
	})

	t.Run("Updates existing watch events", func(t *testing.T) {
		// User 12 is not watching repo 2
		unittest.AssertCount(t, &repo_model.Watch{UserID: 12, RepoID: 2}, 0)

		// First watch with all events
		err := repo_model.WatchRepoWithEvents(db.DefaultContext, 12, 2, repo_model.WatchEventAll)
		require.NoError(t, err)

		watch, err := repo_model.GetWatch(db.DefaultContext, 12, 2)
		require.NoError(t, err)
		assert.Equal(t, repo_model.WatchEventAll, watch.WatchEvents)

		// Update to only releases
		err = repo_model.UpdateWatchEvents(db.DefaultContext, 12, 2, repo_model.WatchEventReleases)
		require.NoError(t, err)

		watch, err = repo_model.GetWatch(db.DefaultContext, 12, 2)
		require.NoError(t, err)
		assert.Equal(t, repo_model.WatchEventReleases, watch.WatchEvents)

		// Clean up
		require.NoError(t, repo_model.WatchRepo(db.DefaultContext, 12, 2, false))
	})
}
