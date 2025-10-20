// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
// AI was used: see https://codeberg.org/forgejo/governance/src/branch/main/AIAgreement.md for AI Agreement
package integration

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	lfs_module "forgejo.org/modules/lfs"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/require"
)

func createLFSFileWithAssertions(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath, data string) api.FileResponse {
	// First, upload the actual content to LFS storage
	hash := sha256.Sum256([]byte(data))
	oid := hex.EncodeToString(hash[:])
	size := int64(len(data))

	// Store the LFS object directly in the LFS store
	contentStore := lfs_module.NewContentStore()
	err := contentStore.Put(lfs_module.Pointer{Oid: oid, Size: size}, strings.NewReader(data))
	require.NoError(t, err, "Failed to store LFS object")

	// Create LFS metadata entry in database
	_, err = git_model.NewLFSMetaObject(db.DefaultContext, repo.ID, lfs_module.Pointer{Oid: oid, Size: size})
	require.NoError(t, err, "Failed to create LFS metadata")

	// Create LFS pointer content
	lfsPointer := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, size)

	createFileOptions := getCreateOptionsFile(lfsPointer, treePath)
	req := NewRequestWithJSON(t, "POST",
		fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath),
		&createFileOptions).AddTokenAuth(token)
	resp := MakeRequest(t, req, http.StatusCreated)

	var fileResponse api.FileResponse
	DecodeJSON(t, resp, &fileResponse)
	require.Equal(t, treePath, fileResponse.Content.Path)

	return fileResponse
}

func verifyLFSFileExistence(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) api.ContentsResponse {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	resp := MakeRequest(t, getReq, http.StatusOK)

	var contentsResponse api.ContentsResponse
	DecodeJSON(t, resp, &contentsResponse)

	// Verify it's an LFS file by checking the content
	require.NotNil(t, contentsResponse.Content, "LFS file should have content")

	// Decode base64 content
	decodedContent, err := base64.StdEncoding.DecodeString(*contentsResponse.Content)
	require.NoError(t, err, "Failed to decode file content")

	content := string(decodedContent)

	// Check for LFS pointer format
	require.Contains(t, content, "version https://git-lfs.github.com/spec/v1", "File should contain LFS version")
	require.Contains(t, content, "oid sha256:", "File should contain LFS oid")
	require.Contains(t, content, "size ", "File should contain LFS size")

	return contentsResponse
}

func verifyLFSFileNotExists(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	MakeRequest(t, getReq, http.StatusNotFound)
}

func isLFSPointerFile(content string) bool {
	// Check if content matches LFS pointer format
	hasVersion := strings.Contains(content, "version https://git-lfs.github.com/spec/v1")
	hasOID := strings.Contains(content, "oid sha256:")
	hasSize := strings.Contains(content, "size ")

	return hasVersion && hasOID && hasSize
}

func assertIsLFSFile(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	resp := MakeRequest(t, getReq, http.StatusOK)

	var contentsResponse api.ContentsResponse
	DecodeJSON(t, resp, &contentsResponse)

	require.NotNil(t, contentsResponse.Content, "File should have content")

	decodedContent, err := base64.StdEncoding.DecodeString(*contentsResponse.Content)
	require.NoError(t, err, "Failed to decode file content")

	content := string(decodedContent)
	require.True(t, isLFSPointerFile(content), "File should be an LFS pointer file")
}

func assertIsNoLFSFile(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	resp := MakeRequest(t, getReq, http.StatusOK)

	var contentsResponse api.ContentsResponse
	DecodeJSON(t, resp, &contentsResponse)

	require.NotNil(t, contentsResponse.Content, "File should have content")

	decodedContent, err := base64.StdEncoding.DecodeString(*contentsResponse.Content)
	require.NoError(t, err, "Failed to decode file content")

	content := string(decodedContent)
	require.False(t, isLFSPointerFile(content), "File should be an LFS pointer file")
}

func TestRecursiveDeleteLFS(t *testing.T) {
	unittest.PrepareTestEnv(t)
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)

		t.Run("Delete LFS file in subdir", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir/file.txt"}
			shouldExist := []bool{
				false,
			}
			treePathsLFS := []string{"testdir/file_LFS.txt"}
			shouldExistLFS := []bool{
				false,
			}
			content := "This is test content for "
			treePathDirDel := "testdir"

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
				assertIsNoLFSFile(t, token, user2, repo1, path)
			}

			for _, path := range treePathsLFS {
				createLFSFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifyLFSFileExistence(t, token, user2, repo1, path)
				assertIsLFSFile(t, token, user2, repo1, path)
			}

			// Verify LFS objects exist in content store before deletion
			contentStore := lfs_module.NewContentStore()
			for _, path := range treePathsLFS {
				hash := sha256.Sum256([]byte(content + path))
				oid := hex.EncodeToString(hash[:])
				exists, err := contentStore.Exists(lfs_module.Pointer{Oid: oid, Size: int64(len(content + path))})
				require.NoError(t, err)
				require.True(t, exists, "LFS object should exist before deletion")
			}

			// Execute delete path command
			deletePathViaUI(t, session, user2, repo1, treePathDirDel)

			// Verify all nested files are removed
			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
					assertIsNoLFSFile(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}

			for i := range treePathsLFS {
				if shouldExistLFS[i] {
					verifyLFSFileExistence(t, token, user2, repo1, treePathsLFS[i])
					assertIsLFSFile(t, token, user2, repo1, treePathsLFS[i])
				} else {
					verifyLFSFileNotExists(t, token, user2, repo1, treePathsLFS[i])
				}
			}
		})
		t.Run("Delete LFS file via root delete", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"testdir/file.txt"}
			shouldExist := []bool{
				false,
			}
			treePathsLFS := []string{"testdir/file_LFS.txt"}
			shouldExistLFS := []bool{
				false,
			}
			content := "This is test content for "
			treePathDirDel := ""

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
				assertIsNoLFSFile(t, token, user2, repo1, path)
			}

			for _, path := range treePathsLFS {
				createLFSFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifyLFSFileExistence(t, token, user2, repo1, path)
				assertIsLFSFile(t, token, user2, repo1, path)
			}

			// Verify LFS objects exist in content store before deletion
			contentStore := lfs_module.NewContentStore()
			for _, path := range treePathsLFS {
				hash := sha256.Sum256([]byte(content + path))
				oid := hex.EncodeToString(hash[:])
				exists, err := contentStore.Exists(lfs_module.Pointer{Oid: oid, Size: int64(len(content + path))})
				require.NoError(t, err)
				require.True(t, exists, "LFS object should exist before deletion")
			}

			// Execute delete path command
			deletePathViaUI(t, session, user2, repo1, treePathDirDel)

			// Verify all nested files are removed
			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
					assertIsNoLFSFile(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}

			for i := range treePathsLFS {
				if shouldExistLFS[i] {
					verifyLFSFileExistence(t, token, user2, repo1, treePathsLFS[i])
					assertIsLFSFile(t, token, user2, repo1, treePathsLFS[i])
				} else {
					verifyLFSFileNotExists(t, token, user2, repo1, treePathsLFS[i])
				}
			}
		})
		t.Run("Delete LFS file with special char", func(t *testing.T) {
			// Create a test file in a directory
			treePaths := []string{"test dir/fi le.txt"}
			shouldExist := []bool{
				false,
			}
			treePathsLFS := []string{"test dir/file LFS.txt"}
			shouldExistLFS := []bool{
				false,
			}
			content := "This is test content for "
			treePathDirDel := "test dir"

			for _, path := range treePaths {
				createFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifiyFileExitence(t, token, user2, repo1, path)
				assertIsNoLFSFile(t, token, user2, repo1, path)
			}

			for _, path := range treePathsLFS {
				createLFSFileWithAssertions(t, token, user2, repo1, path, content+path)
				verifyLFSFileExistence(t, token, user2, repo1, path)
				assertIsLFSFile(t, token, user2, repo1, path)
			}

			// Verify LFS objects exist in content store before deletion
			contentStore := lfs_module.NewContentStore()
			for _, path := range treePathsLFS {
				hash := sha256.Sum256([]byte(content + path))
				oid := hex.EncodeToString(hash[:])
				exists, err := contentStore.Exists(lfs_module.Pointer{Oid: oid, Size: int64(len(content + path))})
				require.NoError(t, err)
				require.True(t, exists, "LFS object should exist before deletion")
			}

			// Execute delete path command
			deletePathViaUI(t, session, user2, repo1, treePathDirDel)

			// Verify all nested files are removed
			// Check result of the delete command
			for i := range treePaths {
				if shouldExist[i] {
					verifiyFileExitence(t, token, user2, repo1, treePaths[i])
					assertIsNoLFSFile(t, token, user2, repo1, treePaths[i])
				} else {
					verifiyFileNoneExitence(t, token, user2, repo1, treePaths[i])
				}
			}

			for i := range treePathsLFS {
				if shouldExistLFS[i] {
					verifyLFSFileExistence(t, token, user2, repo1, treePathsLFS[i])
					assertIsLFSFile(t, token, user2, repo1, treePathsLFS[i])
				} else {
					verifyLFSFileNotExists(t, token, user2, repo1, treePathsLFS[i])
				}
			}
		})
	})
}
