// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"path/filepath"
	"testing"
	"time"

	f3_forge_model "forgejo.org/models/f3/forge"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_MirrorEncryption(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)

	mirror := NewMirror()
	mirror.SetForgeID(forge.ID)
	mirror.SetRemotePath("/forge/users/user1")

	token := "token"
	mirror.SetLocalToken(token)
	mirror.SetRemoteToken(token)
	mirror.encryptTokens()
	assert.NotEqual(t, token, mirror.GetLocalToken())
	assert.NotEqual(t, token, mirror.GetRemoteToken())
	require.NoError(t, mirror.decryptTokens())
	assert.Equal(t, token, mirror.GetLocalToken())
	assert.Equal(t, token, mirror.GetRemoteToken())
}

func TestF3Model_MirrorDatabase(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)

	mirror := NewMirror()
	forgeID := forge.ID
	mirror.SetForgeID(forgeID)
	token := "TOKEN"
	mirror.SetLocalToken(token)
	uid := int64(1345)
	mirror.SetLocalUserID(uid)
	mirror.SetRemoteToken(token)
	remotePath := "/forge/users/user1"
	mirror.SetRemotePath(remotePath)
	localPath := "/forge/users/user2"
	mirror.SetLocalPath(localPath)
	since := timeutil.TimeStampNow()
	mirror.SetSince(since)
	interval, err := time.ParseDuration("24h")
	require.NoError(t, err)
	mirror.SetInterval(interval)
	require.Zero(t, mirror.UpdatedUnix)
	require.Zero(t, mirror.NextUpdateUnix)

	assert.Nil(t, mirror.GetForge())
	mirror.LoadForge(t.Context())
	assert.NotNil(t, mirror.GetForge())
	assert.Equal(t, forge.GetURL(), mirror.GetForge().GetURL())

	var inserted *Mirror
	var updatedUnix timeutil.TimeStamp
	var nextUpdateUnix timeutil.TimeStamp

	t.Run("Upsert", func(t *testing.T) {
		var err error
		inserted, err = Upsert(t.Context(), mirror)
		require.NoError(t, err)
		unittest.AssertCount(t, &Mirror{}, 1)
		assert.NotZero(t, inserted.GetID())
		assert.Equal(t, token, mirror.GetLocalToken())
		assert.Equal(t, uid, mirror.GetLocalUserID())
		assert.Equal(t, token, mirror.GetRemoteToken())
		assert.Equal(t, forgeID, mirror.GetForgeID())
		assert.Equal(t, remotePath, mirror.GetRemotePath())
		assert.Equal(t, localPath, mirror.GetLocalPath())
		assert.Equal(t, since, mirror.GetSince())
		assert.Equal(t, interval, mirror.GetInterval())
		updatedUnix := inserted.UpdatedUnix
		assert.NotZero(t, updatedUnix)
		nextUpdateUnix := inserted.NextUpdateUnix
		assert.NotZero(t, nextUpdateUnix)

		time.Sleep(time.Second) // because the resolution of updatedUnix is 1 second
		same, err := Upsert(t.Context(), inserted)
		require.NoError(t, err)
		unittest.AssertCount(t, &Mirror{}, 1)
		assert.Equal(t, inserted.ID, same.ID)
		assert.Equal(t, updatedUnix, same.UpdatedUnix)
		assert.Equal(t, nextUpdateUnix, inserted.NextUpdateUnix)
		assert.True(t, Equal(inserted, same))
	})
	require.NotNil(t, inserted)

	t.Run("Find", func(t *testing.T) {
		mirrors, err := Find(t.Context(), FindOptions{ForgeID: &forgeID})
		require.NoError(t, err)
		assert.Len(t, mirrors, 1)
		assert.True(t, Equal(inserted, mirrors[0]))
	})

	t.Run("ScheduleNextUpdate", func(t *testing.T) {
		time.Sleep(time.Second) // because the resolution of updatedUnix is 1 second
		modified := NewMirror()
		*modified = *inserted
		modified.ScheduleNextUpdate()
		modifiedInserted, err := Upsert(t.Context(), modified)
		unittest.AssertCount(t, &Mirror{}, 1)
		require.NoError(t, err)
		assert.Equal(t, token, modifiedInserted.GetLocalToken())
		assert.Equal(t, uid, modifiedInserted.GetLocalUserID())
		assert.Equal(t, token, modifiedInserted.GetRemoteToken())
		assert.Greater(t, modifiedInserted.UpdatedUnix, updatedUnix)
		assert.Greater(t, modifiedInserted.NextUpdateUnix, nextUpdateUnix)
		assert.NotEqual(t, modifiedInserted.UpdatedUnix, modifiedInserted.NextUpdateUnix)

		mirrorByID, err := Get(t.Context(), FindOptions{ID: &modified.ID})
		require.NoError(t, err)
		assert.Equal(t, token, mirrorByID.GetLocalToken())
		assert.Equal(t, token, mirrorByID.GetRemoteToken())
		assert.True(t, Equal(modified, mirrorByID))

		mirrorByForgeID, err := Get(t.Context(), FindOptions{ForgeID: &modified.ForgeID})
		require.NoError(t, err)
		assert.Equal(t, token, mirrorByForgeID.GetLocalToken())
		assert.Equal(t, token, mirrorByForgeID.GetRemoteToken())
		assert.True(t, Equal(modified, mirrorByForgeID))
	})
}

func TestF3Model_RemotePathEquivalents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)

	token := "TOKEN"
	remotePath := "/forge/users/user1"
	localPath := "/forge/users/user2"
	uid := int64(4355)
	since := timeutil.TimeStampNow()
	interval, err := time.ParseDuration("24h")
	require.NoError(t, err)

	var redundantIDs []int64
	t.Run("Add (twice to check idempotency)", func(t *testing.T) {
		//
		//   /forge/users/user1/project1
		//   /forge/users/user1/project2
		//
		for _, project := range []string{"project1", "project2"} {
			mirror := NewMirror()
			mirror.SetLocalToken(token)
			mirror.SetLocalUserID(uid)
			mirror.SetRemoteToken(token)
			mirror.SetForgeID(forge.ID)
			mirror.SetRemotePath(filepath.Join(remotePath, project))
			mirror.SetLocalPath(localPath)
			mirror.SetSince(since)
			mirror.SetInterval(interval)
			mirror, err := Upsert(t.Context(), mirror)
			require.NoError(t, err)
			unittest.AssertExistsAndLoadBean(t, &Mirror{ID: mirror.ID})
			redundantIDs = append(redundantIDs, mirror.ID)

			same, err := Upsert(t.Context(), mirror)
			require.NoError(t, err)
			require.Equal(t, mirror.ID, same.ID)
		}
	})

	mirror := NewMirror()
	t.Run("Add (indirectly, via Upsert) /forge/users/user1", func(t *testing.T) {
		//
		// which makes the following, longer, redundant and they are removed
		//   /forge/users/user1/project1
		//   /forge/users/user1/project2
		//
		mirror.SetLocalToken(token)
		mirror.SetLocalUserID(uid)
		mirror.SetRemoteToken(token)
		mirror.SetForgeID(forge.ID)
		mirror.SetRemotePath(remotePath)
		mirror.SetLocalPath(localPath)
		mirror.SetSince(since)
		mirror.SetInterval(interval)
		mirror, err = Upsert(t.Context(), mirror)
		require.NoError(t, err)
		unittest.AssertExistsAndLoadBean(t, &Mirror{ID: mirror.ID})

		for _, redundantID := range redundantIDs {
			unittest.AssertNotExistsBean(t, &Mirror{ID: redundantID})
		}
		unittest.AssertCount(t, &Mirror{}, 1)
	})

	t.Run("Add (indirectly, via Upsert) /forge/users/user1/project3", func(t *testing.T) {
		//
		// does nothing because /forge/users/user1 already exists
		//
		project := NewMirror()
		project.SetLocalToken(token)
		project.SetLocalUserID(uid)
		project.SetRemoteToken(token)
		project.SetForgeID(forge.ID)
		project.SetRemotePath(filepath.Join(remotePath, "project3"))
		project.SetLocalPath(localPath)
		project.SetSince(since)
		project.SetInterval(interval)
		found, err := Upsert(t.Context(), project)
		require.NoError(t, err)
		assert.Equal(t, mirror.ID, found.ID)
		unittest.AssertCount(t, &Mirror{}, 1)
	})

	t.Run("Add (indirectly, via Upsert) /forge/users/user1", func(t *testing.T) {
		//
		// which updates the existing mirror with a new interval
		//
		newInterval := interval + 1
		updatedMirror := NewMirror()
		updatedMirror.SetLocalToken(token)
		updatedMirror.SetRemoteToken(token)
		updatedMirror.SetForgeID(forge.ID)
		updatedMirror.SetRemotePath(remotePath)
		updatedMirror.SetLocalPath(localPath)
		updatedMirror.SetSince(since)
		updatedMirror.SetInterval(newInterval)
		updatedMirror, err = Upsert(t.Context(), updatedMirror)
		require.NoError(t, err)
		require.Equal(t, updatedMirror.ID, mirror.ID)
		assert.Equal(t, newInterval, updatedMirror.Interval)
		unittest.AssertCount(t, &Mirror{}, 1)
	})
}
