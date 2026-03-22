// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repo_test

import (
	"testing"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMirrorsIterateSkipsDisabled(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	now := timeutil.TimeStampNow()
	past := now - 3600

	db.Insert(db.DefaultContext, &repo_model.Mirror{
		RepoID:         99,
		Interval:       time.Hour,
		UpdatedUnix:    past, // I'm overdue for an update
		NextUpdateUnix: past,
		Enabled:        true,
	})

	db.Insert(db.DefaultContext, &repo_model.Mirror{
		RepoID:         100,
		Interval:       time.Hour,
		UpdatedUnix:    past,
		NextUpdateUnix: past,
		Enabled:        false,
	})

	var found []int64
	err := repo_model.MirrorsIterate(db.DefaultContext, 0, func(idx int, bean any) error {
		m := bean.(*repo_model.Mirror)
		found = append(found, m.RepoID)
		return nil
	})
	require.NoError(t, err)

	assert.Contains(t, found, int64(99))
	assert.NotContains(t, found, int64(100))
}

func TestMirrorEnabledExplicit(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	m := &repo_model.Mirror{
		RepoID:   101,
		Interval: time.Hour,
		Enabled:  true,
	}
	require.NoError(t, repo_model.InsertMirror(db.DefaultContext, m))

	retrieved, err := repo_model.GetMirrorByRepoID(db.DefaultContext, 101)
	require.NoError(t, err)
	assert.True(t, retrieved.Enabled)
	assert.Equal(t, 0, retrieved.FailedSyncCount)
}

func TestMirrorFailedSyncCountPersists(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	m := &repo_model.Mirror{
		RepoID:   102,
		Interval: time.Hour,
	}
	require.NoError(t, repo_model.InsertMirror(db.DefaultContext, m))

	// something tragically went wrong
	m.FailedSyncCount = 3
	m.Enabled = false
	require.NoError(t, repo_model.UpdateMirror(db.DefaultContext, m))

	retrieved, err := repo_model.GetMirrorByRepoID(db.DefaultContext, 102)
	require.NoError(t, err)
	assert.Equal(t, 3, retrieved.FailedSyncCount)
	assert.False(t, retrieved.Enabled)

	// problem solved, it should work just fine now
	retrieved.Enabled = true
	retrieved.FailedSyncCount = 0
	require.NoError(t, repo_model.UpdateMirror(db.DefaultContext, retrieved))

	retrieved2, err := repo_model.GetMirrorByRepoID(db.DefaultContext, 102)
	require.NoError(t, err)
	assert.True(t, retrieved2.Enabled)
	assert.Equal(t, 0, retrieved2.FailedSyncCount)
}
