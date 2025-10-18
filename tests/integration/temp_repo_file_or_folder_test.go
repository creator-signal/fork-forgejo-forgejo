// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package integration

import (
	"context"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	files_service "forgejo.org/services/repository/files"

	"github.com/stretchr/testify/require"
)

func TestIsFileOrFolder(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		t.Run("GetTreeEntryType", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir/file.txt"}
			content := "This is test content for "
			branch := "master"

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
			}

			// Create temporary upload repository
			repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, user2.Name, repo1.Name)
			require.NoError(t, err)

			tempRepo, err := files_service.NewTemporaryUploadRepository(ctx, repo)
			require.NoError(t, err)
			defer tempRepo.Close()

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			result, err := tempRepo.GetTreeEntryType("HEAD", "testdir")
			require.NoError(t, err)
			require.Equal(t, "tree", result, "testdir should be a tree (directory)")

			result, err = tempRepo.GetTreeEntryType("HEAD", "testdir/file.txt")
			require.NoError(t, err)
			require.Equal(t, "blob", result, "file.txt should be a blob (file)")

			result, err = tempRepo.GetTreeEntryType("HEAD", "not_existing")
			require.NoError(t, err)
			require.Equal(t, "", result, "not_existing should be unknown")

			result, err = tempRepo.GetTreeEntryType("HEAD", "not_existing/path")
			require.NoError(t, err)
			require.Equal(t, "", result, "not_existing/path should be unknown")

			result, err = tempRepo.GetTreeEntryType("HEAD", "")
			require.NoError(t, err)
			require.Equal(t, "tree", result, "\"\" should be a tree (directory)")
		})
		t.Run("IsDirectory", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir2/file.txt"}
			content := "This is test content for "
			branch := "master"

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
			}

			// Create temporary upload repository
			repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, user2.Name, repo1.Name)
			require.NoError(t, err)

			tempRepo, err := files_service.NewTemporaryUploadRepository(ctx, repo)
			require.NoError(t, err)
			defer tempRepo.Close()

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			result, err := tempRepo.IsDirectory("HEAD", "testdir2")
			require.NoError(t, err)
			require.True(t, result, "testdir2 should be a tree (directory)")

			result, err = tempRepo.IsDirectory("HEAD", "testdir2/file.txt")
			require.NoError(t, err)
			require.False(t, result, "file.txt should be a blob (file)")

			result, err = tempRepo.IsDirectory("HEAD", "not_existing")
			require.NoError(t, err)
			require.False(t, result, "not_existing should be unknown")

			result, err = tempRepo.IsDirectory("HEAD", "not_existing/path")
			require.NoError(t, err)
			require.False(t, result, "not_existing/path should be unknown")

			result, err = tempRepo.IsDirectory("HEAD", "")
			require.NoError(t, err)
			require.True(t, result, "\"\" should be a tree (directory)")
		})
		t.Run("IsFile", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir3/file.txt"}
			content := "This is test content for "
			branch := "master"

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
			}

			// Create temporary upload repository
			repo, err := repo_model.GetRepositoryByOwnerAndName(ctx, user2.Name, repo1.Name)
			require.NoError(t, err)

			tempRepo, err := files_service.NewTemporaryUploadRepository(ctx, repo)
			require.NoError(t, err)
			defer tempRepo.Close()

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			result, err := tempRepo.IsFile("HEAD", "testdir3")
			require.NoError(t, err)
			require.False(t, result, "testdir3 should be a tree (directory)")

			result, err = tempRepo.IsFile("HEAD", "testdir3/file.txt")
			require.NoError(t, err)
			require.True(t, result, "file.txt should be a blob (file)")

			result, err = tempRepo.IsFile("HEAD", "not_existing")
			require.NoError(t, err)
			require.False(t, result, "not_existing should be unknown")

			result, err = tempRepo.IsFile("HEAD", "not_existing/path")
			require.NoError(t, err)
			require.False(t, result, "not_existing/path should be unknown")

			result, err = tempRepo.IsFile("HEAD", "")
			require.NoError(t, err)
			require.False(t, result, "\"\" should be a tree (directory)")
		})
	})
}
