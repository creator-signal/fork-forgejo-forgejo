// Copyright 2025 The Forgejo Authors.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"testing"

	"forgejo.org/models/db"
	migration_tests "forgejo.org/models/gitea_migrations/test"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/schemas"
)

// The main purpose of the WatchSource is to respect explicit user choice and not overwrite that with some automatic system.
type WatchSource bool

const (
	// WatchSourceExplicit means the user explicitly chose to watch certain things (or none or all) of this repo.
	// It means that setting.Service.AutoWatchOnChanges doesn't have an effect on this user for this repo; they explicitly made their choice after all.
	// This mode replaces the old WatchModeDont and WatchModeNormal states.
	WatchSourceExplicit WatchSource = false
	// WatchSourceAutomatic means the user didn't explicitly select whether to watch this repo or not.
	// Instead, the user either doesn't watch the repo because they didn't ever click the watch/unwatch button.
	// Or they do watch the repo but only because the user pushed to the repo and setting.Service.AutoWatchOnChanges is true.
	// When there is no record in the db this is the same as WatchSourceAutomatic combined with all watch selections turned off (i.e., not watching anything).
	// This used to be WatchModeNone.
	// When in this mode the watch selection is never fully deselected.
	// Otherwise there'd be some automatic method to unwatch a repo; which does not exist.
	// This mode replaces the old WatchModeAuto and WatchModeNone states.
	WatchSourceAutomatic WatchSource = true

	// There may not be more modes than the above two.
	// I intend this to be a single bit.
)

// Watch is connection request for receiving repository notification.
type Watch struct {
	ID     int64       `xorm:"pk autoincr"`
	UserID int64       `xorm:"UNIQUE(watch)"`
	RepoID int64       `xorm:"UNIQUE(watch)"`
	Source WatchSource `xorm:"BOOL DEFAULT TRUE"`
	// In the next PR there will be another mode here, choosing the user preset or a custom selection.
	// TODO: figure out whether the user preset should count as watching
	// TODO: (then change the description to IsWatching)

	WatchSelectionIssues       bool `xorm:"BOOL DEFAULT TRUE"`
	WatchSelectionPullRequests bool `xorm:"BOOL DEFAULT TRUE"`
	WatchSelectionReleases     bool `xorm:"BOOL DEFAULT TRUE"`

	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

// GetWatch gets what kind of subscription a user has on a given repository; returns dummy record if none found
func GetWatch(ctx context.Context, userID, repoID int64) (Watch, error) {
	watch := Watch{UserID: userID, RepoID: repoID}
	has, err := db.GetEngine(ctx).Get(&watch)
	if err != nil {
		return watch, err
	}
	if !has {
		watch.Source = WatchSourceAutomatic
		watch.setWatchSelection(WatchNoneSelection)
	}
	return watch, nil
}

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
		watch, err := GetWatch(db.DefaultContext, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceExplicit, watch.Source)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 4, 1)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceExplicit, watch.Source)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 9, 1)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceExplicit, watch.Source)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 8, 1)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceExplicit, watch.Source)
		assert.False(t, watch.WatchSelectionIssues)
		assert.False(t, watch.WatchSelectionPullRequests)
		assert.False(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 11, 1)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceAutomatic, watch.Source)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 4, 2)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceExplicit, watch.Source)
		assert.True(t, watch.WatchSelectionIssues)
		assert.True(t, watch.WatchSelectionPullRequests)
		assert.True(t, watch.WatchSelectionReleases)
	}
	{
		watch, err := GetWatch(db.DefaultContext, 5, 2)
		require.NoError(t, err)
		assert.Equal(t, WatchSourceAutomatic, watch.Source)
		assert.False(t, watch.WatchSelectionIssues)
		assert.False(t, watch.WatchSelectionPullRequests)
		assert.False(t, watch.WatchSelectionReleases)
	}
}
