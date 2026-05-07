// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions_RegisterRunner_Token(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	labels := []string{}
	name := "runner"
	version := "v1.2.3"
	ephemeral := true
	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, &labels, name, version, ephemeral)
	require.NoError(t, err)
	assert.Equal(t, name, runner.Name)
	assert.True(t, runner.Ephemeral)

	ok, _ := auth_model.VerifyHighEntropyToken(token, runner.TokenSalt, runner.TokenHash)
	assert.True(t, ok, "token must be verifiable via VerifyHighEntropyToken")
}

// TestActions_RegisterRunner_TokenUpdate tests that a token's secret is updated
// when a runner already exists and RegisterRunner is called with a token
// parameter whose first 16 bytes match that record but where the last 24 bytes
// do not match.
func TestActions_RegisterRunner_TokenUpdate(t *testing.T) {
	const recordID = 12345678
	oldToken := "7e577e577e577e57feedfacefeedfacefeedface"
	newToken := "7e577e577e577e57deadbeefdeadbeefdeadbeef"
	require.NoError(t, unittest.PrepareTestDatabase())
	before := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	require.Equal(t,
		before.TokenHash, auth_model.HashToken(oldToken, before.TokenSalt),
		"the initial token should match the runner's secret",
	)

	RegisterRunner(db.DefaultContext, before.OwnerID, before.RepoID, newToken, nil, before.Name, before.Version, false)

	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	assert.Equal(t, before.UUID, after.UUID)
	okOld, _ := auth_model.VerifyHighEntropyToken(oldToken, after.TokenSalt, after.TokenHash)
	assert.False(t, okOld, "the old token must no longer verify")
	okNew, _ := auth_model.VerifyHighEntropyToken(newToken, after.TokenSalt, after.TokenHash)
	assert.True(t, okNew, "the new token must verify")
}

func TestActions_RegisterRunner_CreateWithLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	name := "runner"
	version := "v1.2.3"
	ephemeral := true
	labels := []string{"woop", "doop"}
	labelsCopy := labels // labels may be affected by the tested function so we copy them

	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, &labels, name, version, ephemeral)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, ownerID, runner.OwnerID)
	assert.Equal(t, repoID, runner.RepoID)
	assert.Equal(t, name, runner.Name)
	assert.Equal(t, version, runner.Version)
	assert.Equal(t, labelsCopy, runner.AgentLabels)
	assert.Equal(t, ephemeral, runner.Ephemeral)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: runner.ID})
	assert.Equal(t, ownerID, after.OwnerID)
	assert.Equal(t, repoID, after.RepoID)
	assert.Equal(t, name, after.Name)
	assert.Equal(t, version, after.Version)
	assert.Equal(t, labelsCopy, after.AgentLabels)
	assert.Equal(t, ephemeral, after.Ephemeral)
}

func TestActions_RegisterRunner_CreateWithoutLabels(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	ownerID := int64(0)
	repoID := int64(0)
	token := "0123456789012345678901234567890123456789"
	name := "runner"
	version := "v1.2.3"
	ephemeral := true

	runner, err := RegisterRunner(db.DefaultContext, ownerID, repoID, token, nil, name, version, ephemeral)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, ownerID, runner.OwnerID)
	assert.Equal(t, repoID, runner.RepoID)
	assert.Equal(t, name, runner.Name)
	assert.Equal(t, version, runner.Version)
	assert.Equal(t, []string{}, runner.AgentLabels)
	assert.Equal(t, ephemeral, runner.Ephemeral)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: runner.ID})
	assert.Equal(t, ownerID, after.OwnerID)
	assert.Equal(t, repoID, after.RepoID)
	assert.Equal(t, name, after.Name)
	assert.Equal(t, version, after.Version)
	assert.Equal(t, []string{}, after.AgentLabels)
	assert.Equal(t, ephemeral, after.Ephemeral)
}

func TestActions_RegisterRunner_UpdateWithLabels(t *testing.T) {
	const recordID = 12345678
	token := "7e577e577e577e57feedfacefeedfacefeedface"
	require.NoError(t, unittest.PrepareTestDatabase())
	unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	newOwnerID := int64(1)
	newRepoID := int64(1)
	newName := "rennur"
	newVersion := "v4.5.6"
	ephemeral := true
	newLabels := []string{"warp", "darp"}
	labelsCopy := newLabels // labels may be affected by the tested function so we copy them

	runner, err := RegisterRunner(db.DefaultContext, newOwnerID, newRepoID, token, &newLabels, newName, newVersion, ephemeral)
	require.NoError(t, err)

	// Check that the returned record has been updated
	assert.Equal(t, newOwnerID, runner.OwnerID)
	assert.Equal(t, newRepoID, runner.RepoID)
	assert.Equal(t, newName, runner.Name)
	assert.Equal(t, newVersion, runner.Version)
	assert.Equal(t, labelsCopy, runner.AgentLabels)
	assert.Equal(t, ephemeral, runner.Ephemeral)

	// Check that whatever is in the DB has been updated
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	assert.Equal(t, newOwnerID, after.OwnerID)
	assert.Equal(t, newRepoID, after.RepoID)
	assert.Equal(t, newName, after.Name)
	assert.Equal(t, newVersion, after.Version)
	assert.Equal(t, labelsCopy, after.AgentLabels)
	assert.Equal(t, ephemeral, after.Ephemeral)
}

func TestActions_RegisterRunner_UpdateWithoutLabels(t *testing.T) {
	const recordID = 12345678
	token := "7e577e577e577e57feedfacefeedfacefeedface"
	require.NoError(t, unittest.PrepareTestDatabase())
	before := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})

	newOwnerID := int64(1)
	newRepoID := int64(1)
	newName := "rennur"
	newVersion := "v4.5.6"
	ephemeral := true

	runner, err := RegisterRunner(db.DefaultContext, newOwnerID, newRepoID, token, nil, newName, newVersion, ephemeral)
	require.NoError(t, err)

	// Check that the returned record has been updated, except for the labels
	assert.Equal(t, newOwnerID, runner.OwnerID)
	assert.Equal(t, newRepoID, runner.RepoID)
	assert.Equal(t, newName, runner.Name)
	assert.Equal(t, newVersion, runner.Version)
	assert.Equal(t, before.AgentLabels, runner.AgentLabels)
	assert.Equal(t, ephemeral, runner.Ephemeral)

	// Check that whatever is in the DB has been updated, except for the labels
	after := unittest.AssertExistsAndLoadBean(t, &ActionRunner{ID: recordID})
	assert.Equal(t, newOwnerID, after.OwnerID)
	assert.Equal(t, newRepoID, after.RepoID)
	assert.Equal(t, newName, after.Name)
	assert.Equal(t, newVersion, after.Version)
	assert.Equal(t, before.AgentLabels, after.AgentLabels)
	assert.Equal(t, ephemeral, after.Ephemeral)
}
