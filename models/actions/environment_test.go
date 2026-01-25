// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionEnvironment(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("GetEnvironmentByID", func(t *testing.T) {
		env, err := GetEnvironmentByID(t.Context(), 1)
		require.NoError(t, err)
		assert.Equal(t, "production", env.Name)
		assert.EqualValues(t, 1, env.RepoID)

		// Test not found returns custom error type
		_, err = GetEnvironmentByID(t.Context(), 99999)
		require.Error(t, err)
		assert.True(t, IsErrEnvironmentNotFound(err))
	})

	t.Run("GetEnvironmentByName", func(t *testing.T) {
		env, err := GetEnvironmentByName(t.Context(), 0, 1, "production")
		require.NoError(t, err)
		assert.EqualValues(t, 1, env.ID)
		assert.Equal(t, "production", env.Name)

		// Case-insensitive lookup
		env, err = GetEnvironmentByName(t.Context(), 0, 1, "PRODUCTION")
		require.NoError(t, err)
		assert.EqualValues(t, 1, env.ID)

		env, err = GetEnvironmentByName(t.Context(), 0, 1, "Production")
		require.NoError(t, err)
		assert.EqualValues(t, 1, env.ID)

		_, err = GetEnvironmentByName(t.Context(), 0, 1, "nonexistent")
		require.Error(t, err)
		assert.True(t, IsErrEnvironmentNotFound(err))
	})

	t.Run("FindEnvironments", func(t *testing.T) {
		// Find all environments for repo 1
		envs, err := FindEnvironments(t.Context(), FindEnvironmentOptions{RepoID: 1})
		require.NoError(t, err)
		assert.Len(t, envs, 2)

		// Find all environments for repo 2
		envs, err = FindEnvironments(t.Context(), FindEnvironmentOptions{RepoID: 2})
		require.NoError(t, err)
		assert.Len(t, envs, 1)

		// Find by name
		envs, err = FindEnvironments(t.Context(), FindEnvironmentOptions{RepoID: 1, Name: "staging"})
		require.NoError(t, err)
		assert.Len(t, envs, 1)
		assert.Equal(t, "staging", envs[0].Name)

		// Find by name is case-insensitive
		envs, err = FindEnvironments(t.Context(), FindEnvironmentOptions{RepoID: 1, Name: "STAGING"})
		require.NoError(t, err)
		assert.Len(t, envs, 1)
		assert.Equal(t, "staging", envs[0].Name)
	})

	t.Run("CountEnvironments", func(t *testing.T) {
		count, err := CountEnvironments(t.Context(), FindEnvironmentOptions{RepoID: 1})
		require.NoError(t, err)
		assert.EqualValues(t, 2, count)
	})

	t.Run("InsertEnvironment", func(t *testing.T) {
		env, err := InsertEnvironment(t.Context(), 0, 1, "development", "Development environment")
		require.NoError(t, err)
		assert.NotZero(t, env.ID)
		assert.Equal(t, "development", env.Name)
		assert.Equal(t, "Development environment", env.Description)
		assert.EqualValues(t, 0, env.OwnerID)
		assert.EqualValues(t, 1, env.RepoID)

		// Verify it was inserted
		fetched, err := GetEnvironmentByID(t.Context(), env.ID)
		require.NoError(t, err)
		assert.Equal(t, env.Name, fetched.Name)
	})

	t.Run("InsertEnvironment_OwnerIDClearedWhenRepoIDSet", func(t *testing.T) {
		// When both ownerID and repoID are set, ownerID should be cleared to 0
		env, err := InsertEnvironment(t.Context(), 99, 1, "test-precedence", "Test precedence")
		require.NoError(t, err)
		assert.EqualValues(t, 0, env.OwnerID, "OwnerID should be cleared when RepoID is set")
		assert.EqualValues(t, 1, env.RepoID)
	})

	t.Run("InsertEnvironment_WhitespaceTrimmed", func(t *testing.T) {
		env, err := InsertEnvironment(t.Context(), 0, 1, "  whitespace-test  ", "  Description with spaces  ")
		require.NoError(t, err)
		assert.Equal(t, "whitespace-test", env.Name)
		assert.Equal(t, "Description with spaces", env.Description)
	})

	t.Run("InsertEnvironment_NameStoredLowercase", func(t *testing.T) {
		env, err := InsertEnvironment(t.Context(), 0, 1, "MixedCase-ENV", "Test case normalization")
		require.NoError(t, err)
		assert.Equal(t, "mixedcase-env", env.Name, "Name should be stored as lowercase")

		// Should be findable with any case
		fetched, err := GetEnvironmentByName(t.Context(), 0, 1, "MIXEDCASE-ENV")
		require.NoError(t, err)
		assert.Equal(t, env.ID, fetched.ID)
	})

	t.Run("InsertEnvironment_DuplicateNameFails", func(t *testing.T) {
		// First insert should succeed
		_, err := InsertEnvironment(t.Context(), 0, 1, "unique-test", "First")
		require.NoError(t, err)

		// Second insert with same name (different case) should fail
		_, err = InsertEnvironment(t.Context(), 0, 1, "UNIQUE-TEST", "Second")
		require.Error(t, err, "Duplicate name should fail due to UNIQUE constraint")
	})

	t.Run("UpdateEnvironment", func(t *testing.T) {
		env, err := GetEnvironmentByID(t.Context(), 2)
		require.NoError(t, err)

		env.Description = "Updated staging description"
		updated, err := UpdateEnvironment(t.Context(), env)
		require.NoError(t, err)
		assert.True(t, updated)

		// Verify the update
		fetched, err := GetEnvironmentByID(t.Context(), 2)
		require.NoError(t, err)
		assert.Equal(t, "Updated staging description", fetched.Description)
	})

	t.Run("UpdateEnvironment_NotFound", func(t *testing.T) {
		env := &ActionEnvironment{
			ID:      99999,
			OwnerID: 0,
			RepoID:  1,
			Name:    "nonexistent",
		}
		updated, err := UpdateEnvironment(t.Context(), env)
		require.NoError(t, err)
		assert.False(t, updated, "Update should return false when environment not found")
	})

	t.Run("DeleteEnvironment", func(t *testing.T) {
		// Create a new environment to delete
		env, err := InsertEnvironment(t.Context(), 0, 1, "temp", "Temporary environment")
		require.NoError(t, err)

		deleted, err := DeleteEnvironment(t.Context(), env.ID, env.OwnerID, env.RepoID)
		require.NoError(t, err)
		assert.True(t, deleted)

		// Verify it was deleted
		_, err = GetEnvironmentByID(t.Context(), env.ID)
		require.Error(t, err)
		assert.True(t, IsErrEnvironmentNotFound(err))
	})

	t.Run("DeleteEnvironment_NotFound", func(t *testing.T) {
		deleted, err := DeleteEnvironment(t.Context(), 99999, 0, 1)
		require.NoError(t, err)
		assert.False(t, deleted, "Delete should return false when environment not found")
	})

	t.Run("DeleteEnvironment_WrongOwnership", func(t *testing.T) {
		// Try to delete with wrong owner/repo - should fail
		env, err := GetEnvironmentByID(t.Context(), 1)
		require.NoError(t, err)

		// Use wrong repoID
		deleted, err := DeleteEnvironment(t.Context(), env.ID, env.OwnerID, 999)
		require.NoError(t, err)
		assert.False(t, deleted, "Delete should fail with wrong ownership")

		// Verify it still exists
		_, err = GetEnvironmentByID(t.Context(), 1)
		require.NoError(t, err)
	})
}
