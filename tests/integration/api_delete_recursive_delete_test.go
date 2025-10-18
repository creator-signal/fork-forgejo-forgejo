// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
)

func deletePathViaAPI(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	// Get the SHA for the file/directory
	deleteFileOptions := getDeleteFileOptions()
	deleteFileOptions.BranchName = "master"
	deleteFileOptions.SHA = "0000000000000000000000000000000000000000" // Fake SHA

	req := NewRequestWithJSON(t, "DELETE",
		fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath),
		&deleteFileOptions).AddTokenAuth(token)

	resp := MakeRequest(t, req, http.StatusOK)
	var fileResponse api.FileResponse
	DecodeJSON(t, resp, &fileResponse)
}

func TestAPIRecursiveDeleteSubSub(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "dir2/dir3"

		shouldExist := []bool{
			true,
			true,
			false,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// Execute delete path command
		deletePathViaAPI(t, token2, user2, repo1, treePathDirDel)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteSub(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "dir2"

		shouldExist := []bool{
			true,
			false,
			false,
			false,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// Execute delete path command
		deletePathViaAPI(t, token2, user2, repo1, treePathDirDel)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

// Deleting root is not supported via API
func TestAPIRecursiveDeleteRoot(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := ""

		shouldExist := []bool{
			true,
			true,
			true,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// Execute delete path command
		deleteFileOptions := getDeleteFileOptions()
		deleteFileOptions.BranchName = "master"
		deleteFileOptions.SHA = "0000000000000000000000000000000000000000" // Fake SHA

		req := NewRequestWithJSON(t, "DELETE",
			fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePathDirDel),
			&deleteFileOptions).AddTokenAuth(token2)

		MakeRequest(t, req, http.StatusMethodNotAllowed)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteAnonymous(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := ""

		shouldExist := []bool{
			true,
			true,
			true,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// trying to delete the files without permission
		deleteFileOptions := getDeleteFileOptions()
		deleteFileOptions.BranchName = "master"
		deleteFileOptions.SHA = "0000000000000000000000000000000000000000" // Fake SHA

		req := NewRequestWithJSON(t, "DELETE",
			fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePathDirDel),
			&deleteFileOptions).AddTokenAuth(token2)

		MakeRequest(t, req, http.StatusMethodNotAllowed)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteOther(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := ""

		shouldExist := []bool{
			true,
			true,
			true,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})       // not the owner
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// user4
		session4 := loginUser(t, user4.Name)
		token4 := getTokenForLoggedInUser(t, session4, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		deleteFileOptions := getDeleteFileOptions()
		deleteFileOptions.BranchName = "master"
		deleteFileOptions.SHA = "0000000000000000000000000000000000000000" // Fake SHA

		req := NewRequestWithJSON(t, "DELETE",
			fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePathDirDel),
			&deleteFileOptions).AddTokenAuth(token4)

		MakeRequest(t, req, http.StatusMethodNotAllowed)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteDirColab(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "dir2"

		shouldExist := []bool{
			true,
			false,
			false,
			false,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		user3 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 3})       // owner of the repo3
		repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}) // public repo of user3

		// Get user2's token
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files by user3
		for i := range treePaths {
			createFileWithAssertions(t, token, user3, repo3, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token, user3, repo3, treePaths[i])
		}

		// Execute delete path command by user2
		deletePathViaAPI(t, token, user3, repo3, treePathDirDel)

		// Check result of the delete command by user3
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token, user3, repo3, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token, user3, repo3, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteNoneExist(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "does_not_exist"

		shouldExist := []bool{
			true,
			true,
			true,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		deleteFileOptions := getDeleteFileOptions()
		deleteFileOptions.BranchName = "master"
		deleteFileOptions.SHA = "0000000000000000000000000000000000000000" // Fake SHA

		req := NewRequestWithJSON(t, "DELETE",
			fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user2.Name, repo1.Name, treePathDirDel),
			&deleteFileOptions).AddTokenAuth(token2)

		MakeRequest(t, req, http.StatusNotFound)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}

func TestAPIRecursiveDeleteSpecialChars(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"there is a space in the folder name/file1.txt",
			"there is a space in the folder name1/file2.txt",
		}

		treePathDirDel := "there is a space in the folder name"

		shouldExist := []bool{
			false,
			true,
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		for i := range treePaths {
			createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
		}

		// Execute delete path command
		deletePathViaAPI(t, token2, user2, repo1, treePathDirDel)

		// Check result of the delete command
		for i := range treePaths {
			if shouldExist[i] {
				verifiyFileExitence(t, token2, user2, repo1, treePaths[i])
			} else {
				verifiyFileNoneExitence(t, token2, user2, repo1, treePaths[i])
			}
		}
	})
}
