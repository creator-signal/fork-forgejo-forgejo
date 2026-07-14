// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"bytes"
	"testing"
	"time"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/lfs"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/storage"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunManualGCForRepo(t *testing.T) {
	unittest.PrepareTestEnv(t)
	require.NoError(t, storage.Init())

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "lfs")
	require.NoError(t, err)

	require.NoError(t, RunManualGCForRepo(t.Context(), repo))

	repoAfter, err := repo_model.GetRepositoryByID(db.DefaultContext, repo.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, repoAfter.GitSize, int64(0))
}

func TestRunManualGCForRepo_RemovesOrphanedCommit(t *testing.T) {
	unittest.PrepareTestEnv(t)
	defer test.MockVariableValue(&setting.LFS.StartServer, true)()
	require.NoError(t, storage.Init())
	defer test.MockVariableValue(&setting.Git.GCArgs, []string{"--prune=now"})()

	author := &git.Signature{Name: "Test", Email: "test@test.com", When: time.Now()}

	repo, err := repo_model.GetRepositoryByOwnerAndName(db.DefaultContext, "user2", "lfs")
	require.NoError(t, err)

	gitRepo, err := git.OpenRepository(t.Context(), repo.RepoPath())
	require.NoError(t, err)
	defer gitRepo.Close()

	headCommit, err := gitRepo.GetBranchCommit("master")
	require.NoError(t, err)

	firstID, err := gitRepo.CommitTree(author, author, &headCommit.Tree, git.CommitTreeOpts{
		Parents:   []string{headCommit.ID.String()},
		Message:   "initial commit",
		NoGPGSign: true,
	})
	require.NoError(t, err)
	require.NoError(t, gitRepo.SetReference("refs/heads/master", firstID.String()))

	// run GC make sure gitSize is > 0
	// to verify the initial commit is present
	require.NoError(t, RunManualGCForRepo(t.Context(), repo))
	repoAfterFirst, err := repo_model.GetRepositoryByID(db.DefaultContext, repo.ID)
	require.NoError(t, err)
	assert.Positive(t, repoAfterFirst.GitSize)

	// Amend commit, overwriting initial commit reference
	amendedID, err := gitRepo.CommitTree(author, author, &headCommit.Tree, git.CommitTreeOpts{
		Parents:   []string{headCommit.ID.String()},
		Message:   "amended commit",
		NoGPGSign: true,
	})
	require.NoError(t, err)
	require.NoError(t, gitRepo.SetReference("refs/heads/master", amendedID.String()))

	// Both commits exist before GC cleans up
	assert.True(t, gitRepo.IsObjectExist(firstID.String()))
	assert.True(t, gitRepo.IsObjectExist(amendedID.String()))

	// Run GC again, now the first commit should be pruned
	require.NoError(t, RunManualGCForRepo(t.Context(), repo))
	repoAfterSecond, err := repo_model.GetRepositoryByID(db.DefaultContext, repo.ID)
	require.NoError(t, err)
	assert.Less(t, repoAfterSecond.GitSize, repoAfterFirst.GitSize)

	// Open a fresh repo handle to verify first commit is gone
	// the first git gitRepo bariable is stale and will not display changes
	gitRepo2, err := git.OpenRepository(t.Context(), repo.RepoPath())
	require.NoError(t, err)
	defer gitRepo2.Close()

	assert.False(t, gitRepo2.IsObjectExist(firstID.String()))
	assert.True(t, gitRepo2.IsObjectExist(amendedID.String()))
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
