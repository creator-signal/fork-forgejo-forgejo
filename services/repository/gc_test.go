// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"bytes"
	"testing"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunManualGCForRepo(t *testing.T) {
	unittest.PrepareTestEnv(t)
	defer test.MockVariableValue(&setting.LFS.StartServer, true)()
	require.NoError(t, storage.Init())

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "lfs")
	require.NoError(t, err)

	require.NoError(t, RunManualGCForRepo(t.Context(), repo))

	repoAfter, err := repo_model.GetRepositoryByID(db.DefaultContext, repo.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, repoAfter.GitSize, int64(0))
}

func TestRunManualGCForRepo_RemovesOrphanedLFSObjects(t *testing.T) {
	unittest.PrepareTestEnv(t)
	defer test.MockVariableValue(&setting.LFS.StartServer, true)()
	require.NoError(t, storage.Init())

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "lfs")
	require.NoError(t, err)

	content := []byte("manual-gc-test-orphan")
	pointer, err := lfs.GeneratePointer(bytes.NewReader(content))
	require.NoError(t, err)
	_, err = git_model.NewLFSMetaObject(db.DefaultContext, repo.ID, pointer)
	require.NoError(t, err)
	store := lfs.NewContentStore()
	require.NoError(t, store.Put(pointer, bytes.NewReader(content)))

	require.NoError(t, RunManualGCForRepo(t.Context(), repo))

	_, err = git_model.GetLFSMetaObjectByOid(db.DefaultContext, repo.ID, pointer.Oid)
	require.ErrorIs(t, err, git_model.ErrLFSObjectNotExist)
}
