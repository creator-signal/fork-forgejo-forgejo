// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package mirror

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	f3_forge_model "forgejo.org/models/f3/forge"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/timeutil"
	permissions_errors "forgejo.org/services/permissions/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestF3Model_MirrorAccessors(t *testing.T) {
	mirror := NewMirror()

	id := int64(1324)
	mirror.SetID(id)
	assert.Equal(t, id, mirror.GetID())

	token := "TOKEN"
	mirror.SetToken(token)
	assert.Equal(t, token, mirror.GetToken())

	forgeID := int64(888)
	mirror.SetForgeID(forgeID)
	assert.Equal(t, forgeID, mirror.GetForgeID())

	fromPath := "/forge/users/user1"
	mirror.SetFromPath(fromPath)
	assert.Equal(t, fromPath, mirror.GetFromPath())

	toPath := "/forge/users/user2"
	mirror.SetToPath(toPath)
	assert.Equal(t, toPath, mirror.GetToPath())

	since := timeutil.TimeStampNow()
	mirror.SetSince(since)
	assert.Equal(t, since, mirror.GetSince())

	interval, err := time.ParseDuration("24h")
	require.NoError(t, err)
	mirror.SetInterval(interval)
	assert.Equal(t, interval, mirror.GetInterval())

	sendNotifications := true
	mirror.SetSendNotifications(sendNotifications)
	assert.Equal(t, sendNotifications, mirror.GetSendNotifications())

	for _, testCase := range []struct {
		name string
		err  error
	}{
		{
			name: "NotFound",
			err:  permissions_errors.NewNotFound("Message"),
		},
		{
			name: "Server",
			err:  permissions_errors.NewServer("Message"),
		},
		{
			name: "Forbidden",
			err:  permissions_errors.NewForbidden("Message"),
		},
	} {
		t.Run("Error "+testCase.name, func(t *testing.T) {
			mirror := NewMirror()
			mirror.SetError(testCase.err)
			assert.Equal(t, testCase.name, mirror.Err)
			assert.Equal(t, "Message", mirror.ErrMessage)
			err := mirror.GetError()
			assert.Equal(t, testCase.err, err)
		})
	}
	t.Run("Error nil", func(t *testing.T) {
		mirror := NewMirror()
		mirror.SetError(nil)
		assert.Equal(t, NoError, mirror.Err)
		assert.Empty(t, mirror.ErrMessage)
		assert.NoError(t, mirror.GetError())
	})
	t.Run("Error other", func(t *testing.T) {
		mirror := NewMirror()
		message := "OTHER"
		mirror.SetError(errors.New(message))
		assert.Equal(t, OtherError, mirror.Err)
		assert.Equal(t, message, mirror.ErrMessage)
		err := mirror.GetError()
		assert.Equal(t, message, err.Error())
	})
}

func TestF3Model_MirrorEncryption(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)

	mirror := NewMirror()
	mirror.SetForgeID(forge.ID)
	mirror.SetFromPath("/forge/users/user1")

	token := "token"
	mirror.SetToken(token)
	encryptedToken := mirror.encryptToken()
	mirror.SetToken(string(encryptedToken))
	decryptedToken, err := mirror.decryptToken()
	require.NoError(t, err)
	assert.Equal(t, token, string(decryptedToken))
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
	mirror.SetToken(token)
	fromPath := "/forge/users/user1"
	mirror.SetFromPath(fromPath)
	toPath := "/forge/users/user2"
	mirror.SetToPath(toPath)
	since := timeutil.TimeStampNow()
	mirror.SetSince(since)
	interval, err := time.ParseDuration("24h")
	require.NoError(t, err)
	mirror.SetInterval(interval)
	require.Zero(t, mirror.UpdatedUnix)
	require.Zero(t, mirror.NextUpdateUnix)

	inserted, err := Upsert(t.Context(), mirror)
	require.NoError(t, err)
	unittest.AssertCount(t, &Mirror{}, 1)
	assert.NotZero(t, inserted.GetID())
	assert.Equal(t, token, mirror.GetToken())
	assert.Equal(t, forgeID, mirror.GetForgeID())
	assert.Equal(t, fromPath, mirror.GetFromPath())
	assert.Equal(t, toPath, mirror.GetToPath())
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

	{
		mirrors, err := Find(t.Context(), FindOptions{ForgeID: &forgeID})
		require.NoError(t, err)
		assert.Len(t, mirrors, 1)
		assert.True(t, Equal(inserted, mirrors[0]))
	}

	time.Sleep(time.Second) // because the resolution of updatedUnix is 1 second
	modified := NewMirror()
	*modified = *inserted
	modified.ScheduleNextUpdate()
	modifiedInserted, err := Upsert(t.Context(), modified)
	unittest.AssertCount(t, &Mirror{}, 1)
	require.NoError(t, err)
	assert.Equal(t, token, modifiedInserted.GetToken())
	assert.Greater(t, modifiedInserted.UpdatedUnix, updatedUnix)
	assert.Greater(t, modifiedInserted.NextUpdateUnix, nextUpdateUnix)
	assert.NotEqual(t, modifiedInserted.UpdatedUnix, modifiedInserted.NextUpdateUnix)

	mirrorByID, err := Get(t.Context(), FindOptions{ID: &modified.ID})
	require.NoError(t, err)
	assert.Equal(t, token, mirrorByID.GetToken())
	assert.True(t, Equal(modified, mirrorByID))

	mirrorByForgeID, err := Get(t.Context(), FindOptions{ForgeID: &modified.ForgeID})
	require.NoError(t, err)
	assert.Equal(t, token, mirrorByForgeID.GetToken())
	assert.True(t, Equal(modified, mirrorByForgeID))
}

func TestF3Model_FromPathEquivalents(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	forge := f3_forge_model.NewForge()
	forge.SetURL("URL")
	forge, err := f3_forge_model.Upsert(t.Context(), forge)
	require.NoError(t, err)

	token := "TOKEN"
	fromPath := "/forge/users/user1"
	toPath := "/forge/users/user2"
	since := timeutil.TimeStampNow()
	interval, err := time.ParseDuration("24h")
	require.NoError(t, err)

	//
	// Add (twice to check idempotency)
	//   /forge/users/user1/project1
	//   /forge/users/user1/project2
	//
	var redundantIDs []int64
	for _, project := range []string{"project1", "project2"} {
		mirror := NewMirror()
		mirror.SetToken(token)
		mirror.SetForgeID(forge.ID)
		mirror.SetFromPath(filepath.Join(fromPath, project))
		mirror.SetToPath(toPath)
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

	//
	// Add (indirectly, via Upsert)
	//   /forge/users/user1
	// which makes the following, longer, redundant and they are removed
	//   /forge/users/user1/project1
	//   /forge/users/user1/project2
	//
	mirror := NewMirror()
	mirror.SetToken(token)
	mirror.SetForgeID(forge.ID)
	mirror.SetFromPath(fromPath)
	mirror.SetToPath(toPath)
	mirror.SetSince(since)
	mirror.SetInterval(interval)
	mirror, err = Upsert(t.Context(), mirror)
	require.NoError(t, err)
	unittest.AssertExistsAndLoadBean(t, &Mirror{ID: mirror.ID})

	for _, redundantID := range redundantIDs {
		unittest.AssertNotExistsBean(t, &Mirror{ID: redundantID})
	}
	unittest.AssertCount(t, &Mirror{}, 1)

	//
	// Add (indirectly, via Upsert)
	//   /forge/users/user1/project3
	// does nothing because /forge/users/user1 already exists
	//
	project := NewMirror()
	project.SetToken(token)
	project.SetForgeID(forge.ID)
	project.SetFromPath(filepath.Join(fromPath, "project3"))
	project.SetToPath(toPath)
	project.SetSince(since)
	project.SetInterval(interval)
	project, err = Upsert(t.Context(), project)
	require.NoError(t, err)
	assert.Nil(t, project)
	unittest.AssertCount(t, &Mirror{}, 1)

	//
	// Add (indirectly, via Upsert)
	//   /forge/users/user1
	// which updates the existing mirror with a new interval
	//
	newInterval := interval + 1
	updatedMirror := NewMirror()
	updatedMirror.SetToken(token)
	updatedMirror.SetForgeID(forge.ID)
	updatedMirror.SetFromPath(fromPath)
	updatedMirror.SetToPath(toPath)
	updatedMirror.SetSince(since)
	updatedMirror.SetInterval(newInterval)
	updatedMirror, err = Upsert(t.Context(), updatedMirror)
	require.NoError(t, err)
	require.Equal(t, updatedMirror.ID, mirror.ID)
	assert.Equal(t, newInterval, updatedMirror.Interval)
	unittest.AssertCount(t, &Mirror{}, 1)
}
