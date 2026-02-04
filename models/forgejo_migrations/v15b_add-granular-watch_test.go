// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"testing"

	"forgejo.org/models/db"
	migration_tests "forgejo.org/models/gitea_migrations/test"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

func Test_addGranularWatchColumnsAndDropModeColumn(t *testing.T) {
	// copy of old code //
	type WatchMode uint8
	type Watch struct {
		ID     int64     `xorm:"pk autoincr"`
		UserID int64     `xorm:"UNIQUE(watch)"`
		RepoID int64     `xorm:"UNIQUE(watch)"`
		Mode   WatchMode `xorm:"SMALLINT NOT NULL DEFAULT 1"`

		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
	}
	// end copy of old code //

	// Prepare TestEnv
	x, deferable := migration_tests.PrepareTestEnv(t, 0,
		new(Watch),
	)
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	// test for expected results
	getColumn := func(tn, co string) *schemas.Column {
		tables, err := x.DBMetas()
		require.NoError(t, err)
		var table *schemas.Table
		for _, elem := range tables {
			if elem.Name == tn {
				table = elem
				break
			}
		}
		return table.GetColumn(co)
	}

	require.NotNil(t, getColumn("watch", "mode"))
	_, err := x.Table("watch").Count()
	require.NoError(t, err)

	require.NoError(t, addGranularWatchColumnsAndDropModeColumn(x))

	require.Nil(t, getColumn("watch", "mode"))
	require.NotNil(t, getColumn("watch", "watch_selection_issues"))
	require.NotNil(t, getColumn("watch", "watch_selection_pull_requests"))
	require.NotNil(t, getColumn("watch", "watch_selection_releases"))
	cnt2, err := x.Table("watch").Count()
	require.NoError(t, err)
	require.Equal(t, int64(7), cnt2)

	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceExplicit)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 4, 1)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceExplicit)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 9, 1)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceExplicit)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 8, 1)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceExplicit)
		assert.False(t, watch.WatchSelectionIssues)
		assert.False(t, watch.WatchSelectionPullRequests)
		assert.False(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 11, 1)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceAutomatic)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 4, 2)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceExplicit)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := repo_model.GetWatch(db.DefaultContext, 5, 2)
		require.NoError(t, err)
		assert.Equal(t, watch.Source, repo_model.WatchSourceAutomatic)
		assert.False(t, watch.WatchSelectionIssues)
		assert.False(t, watch.WatchSelectionPullRequests)
		assert.False(t, watch.WatchSelectionReleases)
	}
}
