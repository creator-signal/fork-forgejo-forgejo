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

func TestRemoveDirectoryRecursively(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		t.Run("RemoveSingleDirectory", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir/file.txt"}
			content := "This is test content for "
			treePathDirDel := "testdir"

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

			// Define branch and commit options
			branch := "master"
			message := "Remove testdir"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Verify all nested files are removed
			for _, path := range treePaths {
				verifiyFileNoneExitence(t, token, user2, repo1, path)
			}
		})
		t.Run("RemoveNestedDirectories", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"parent/file1.txt",
				"parent/child/file2.txt",
				"parent/child/grandchild/file3.txt",
			}
			content := "This is test content for "
			treePathDirDel := "parent"

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

			// Define branch and commit options
			branch := "master"
			message := "Remove parent directory"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Verify all nested files are removed
			for _, path := range treePaths {
				verifiyFileNoneExitence(t, token, user2, repo1, path)
			}
		})
		t.Run("RemoveNonExistentDirectory", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"parent/file1.txt",
				"parent/child/file2.txt",
				"parent/child/grandchild/file3.txt",
			}
			content := "This is test content for "
			treePathDirDel := "nonexistent"

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

			// Define branch and commit options
			branch := "master"

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.Error(t, err, "Should fail when removing non-existent directory")
			require.Contains(t, err.Error(), "Unable to remove directory")
		})
		t.Run("RemoveDirectoryWithSpecialCharacters", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"dir with spaces/file.txt"}
			content := "This is test content for "
			treePathDirDel := "dir with spaces"

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

			// Define branch and commit options
			branch := "master"
			message := "Remove directory with spaces"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Verify all nested files are removed
			for _, path := range treePaths {
				verifiyFileNoneExitence(t, token, user2, repo1, path)
			}
		})
		t.Run("RemoveMultipleFilesInDirectory", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"multidir/file1.txt",
				"multidir/file2.txt",
				"multidir/file3.txt",
			}
			content := "This is test content for "
			treePathDirDel := "multidir"

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

			// Define branch and commit options
			branch := "master"
			message := "Remove directory with multiple files"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Verify all nested files are removed
			for _, path := range treePaths {
				verifiyFileNoneExitence(t, token, user2, repo1, path)
			}
		})

		t.Run("RemovePartialPath", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"root/middle/deep/file.txt",
				"root/middle/file2.txt",
				"root/file3.txt",
			}
			content := "This is test content for "
			treePathDirDel := "root/middle"

			shouldExist := []bool{
				false,
				false,
				true,
			}

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

			// Define branch and commit options
			branch := "master"
			message := "Should remove subdirectory root/middle"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}
		})
		t.Run("RemoveEmptyDirectoryWithGitkeep", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"emptydir/.gitkeep",
			}
			content := "This is test content for "
			treePathDirDel := "emptydir"

			shouldExist := []bool{
				false,
			}

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

			// Define branch and commit options
			branch := "master"
			message := "Should remove directory with only .gitkeep"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}
		})
		t.Run("RemoveRootDirectory", func(t *testing.T) {
			// Create test files in a directory
			treePaths := []string{
				"rootkill/middle/deep/file.txt",
				"rootkill/middle/file2.txt",
				"rootkill/file3.txt",
				"file4.txt",
			}
			content := "This is test content for "
			treePathDirDel := "."

			shouldExist := []bool{
				false,
				false,
				false,
				false,
			}

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

			// Define branch and commit options
			branch := "master"
			message := "Remove everything"

			signoff := false

			author := user2
			committer := user2

			// Clone the branch
			err = tempRepo.Clone(branch, false)
			require.NoError(t, err)

			// Set up the index
			err = tempRepo.SetDefaultIndex()
			require.NoError(t, err)

			err = tempRepo.RefreshIndex()
			require.NoError(t, err)

			// Remove the directory recursively
			err = tempRepo.RemoveDirectoryRecursively(treePathDirDel)
			require.NoError(t, err)

			// Write the tree
			treeHash, err := tempRepo.WriteTree()
			require.NoError(t, err)

			// Commit the tree
			commitHash, err := tempRepo.CommitTree("HEAD", author, committer, treeHash, message, signoff)
			require.NoError(t, err)

			// Push to the new branch
			err = tempRepo.Push(user2, commitHash, branch)
			require.NoError(t, err)

			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}
		})
	})
}
