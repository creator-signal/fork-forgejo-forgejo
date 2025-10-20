// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later
// AI was used: see https://codeberg.org/forgejo/governance/src/branch/main/AIAgreement.md for AI Agreement
package integration

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
)

func getCreateOptionsFile(content, message string) api.CreateFileOptions {
	contentEncoded := base64.StdEncoding.EncodeToString([]byte(content))
	return api.CreateFileOptions{
		FileOptions: api.FileOptions{
			BranchName:    "master",
			NewBranchName: "master",
			Message:       message,
			Author: api.Identity{
				Name:  "Anne Doe",
				Email: "annedoe@example.com",
			},
			Committer: api.Identity{
				Name:  "John Doe",
				Email: "johndoe@example.com",
			},
			Dates: api.CommitDateOptions{
				Author:    time.Unix(946684810, 0),
				Committer: time.Unix(978307190, 0),
			},
		},
		ContentBase64: contentEncoded,
	}
}

func createFileWithAssertions(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath, data string) api.FileResponse {
	createFileOptions := getCreateOptionsFile(data, treePath)

	req := NewRequestWithJSON(t, "POST",
		fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath),
		&createFileOptions).AddTokenAuth(token)

	resp := MakeRequest(t, req, http.StatusCreated)

	var fileResponse api.FileResponse
	DecodeJSON(t, resp, &fileResponse)

	assert.Equal(t, fileResponse.Content.Path, treePath)
	assert.EqualValues(t, len(data), fileResponse.Content.Size)

	return fileResponse
}

func verifiyFileExitence(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	MakeRequest(t, getReq, http.StatusOK)
}

func verifiyFileNoneExitence(t *testing.T, token string, user *user_model.User, repo *repo_model.Repository, treePath string) {
	getReq := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/contents/%s", user.Name, repo.Name, treePath)).AddTokenAuth(token)
	MakeRequest(t, getReq, http.StatusNotFound)
}

func deletePathViaUI(t *testing.T, session *TestSession, user *user_model.User, repo *repo_model.Repository, treePath string) {
	deletePath := fmt.Sprintf("/%s/%s/_delete/master/%s", user.Name, repo.Name, treePath)

	// Get the delete page to obtain CSRF token
	req := NewRequest(t, "GET", deletePath)
	session.MakeRequest(t, req, http.StatusOK)

	csrf := GetCSRF(t, session, deletePath)

	// Prepare commit form
	commitForm := map[string]string{
		"_csrf":          csrf,
		"commit_summary": "Delete",
		"commit_message": "",
		"commit_choice":  "direct",
		"commit_mail_id": "-1",
	}

	// POST to delete the path
	postReq := NewRequestWithValues(t, "POST", deletePath, commitForm)
	postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

	// Follow redirect if present
	if redirectLocation := postResp.Header().Get("Location"); redirectLocation != "" {
		verifyReq := NewRequest(t, "GET", redirectLocation)
		session.MakeRequest(t, verifyReq, http.StatusOK)
	}
}

func TestRecursiveDeleteSubSub(t *testing.T) {
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
		deletePathViaUI(t, session, user2, repo1, treePathDirDel)

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

func TestRecursiveDeleteSub(t *testing.T) {
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
		deletePathViaUI(t, session, user2, repo1, treePathDirDel)

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

func TestRecursiveDeleteRoot(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := ""

		shouldExist := []bool{
			false,
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
		deletePathViaUI(t, session, user2, repo1, treePathDirDel)

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

func TestRecursiveDeleteAnonymous(t *testing.T) {
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
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete/master/%s", user2.Name, repo1.Name, treePathDirDel))
		MakeRequest(t, req, http.StatusSeeOther)

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

func TestRecursiveDeleteOther(t *testing.T) {
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
		session = loginUser(t, user4.Name)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/_delete/master/%s", user2.Name, repo1.Name, treePathDirDel))
		session.MakeRequest(t, req, http.StatusNotFound)

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

func TestRecursiveDeleteRootColab(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := ""

		shouldExist := []bool{
			false,
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
		deletePathViaUI(t, session, user3, repo3, treePathDirDel)

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

func TestIfDeleteButtonIsThereUser(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo of user3

		// Get user2's token
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files by user3
		for i := range treePaths {
			createFileWithAssertions(t, token, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token, user2, repo1, treePaths[i])
		}

		// Fetch the page (/dir1)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/dir2", user2.Name, repo1.Name))
		resp := session.MakeRequest(t, req, http.StatusOK)

		// Parse HTML response
		doc := NewHTMLParser(t, resp.Body)

		// Check if delete button exists -> should exist
		deleteButton := doc.Find("a[href*='/_delete/']").First()
		if deleteButton.Length() == 0 {
			t.Error("Delete folder button not found")
		}

		// Fetch the page ()
		req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master", user2.Name, repo1.Name))
		resp = session.MakeRequest(t, req, http.StatusOK)

		// Parse HTML response
		doc = NewHTMLParser(t, resp.Body)

		// Check if delete button exists -> should exist
		deleteButton = doc.Find("a[href*='/_delete/']").First()
		if deleteButton.Length() == 0 {
			t.Error("Delete path button not found")
		}
	})
}

func TestIfDeleteButtonIsThereAnonymous(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo of user3

		// Get user2's token
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files by user3
		for i := range treePaths {
			createFileWithAssertions(t, token, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			verifiyFileExitence(t, token, user2, repo1, treePaths[i])
		}

		// Fetch the page (/dir1)
		req := NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master/dir2", user2.Name, repo1.Name))
		resp := MakeRequest(t, req, http.StatusOK)

		// Parse HTML response
		doc := NewHTMLParser(t, resp.Body)

		// Check if delete button exists -> should NOT exist
		deleteButton := doc.Find("a[href*='/_delete/']").First()
		if deleteButton.Length() != 0 {
			t.Error("Delete path button found")
		}

		// Fetch the page ()
		req = NewRequest(t, "GET", fmt.Sprintf("/%s/%s/src/branch/master", user2.Name, repo1.Name))
		resp = MakeRequest(t, req, http.StatusOK)

		// Parse HTML response
		doc = NewHTMLParser(t, resp.Body)

		// Check if delete button xists -> should NOT exist
		deleteButton = doc.Find("a[href*='/_delete/']").First()
		if deleteButton.Length() != 0 {
			t.Error("Delete path button found")
		}
	})
}

func TestRecursiveDeleteNoneExist(t *testing.T) {
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

		deletePath := fmt.Sprintf("/%s/%s/_delete/master/%s", user2.Name, repo1.Name, treePathDirDel)

		// Get the delete page to obtain CSRF token
		req := NewRequest(t, "GET", deletePath)
		session.MakeRequest(t, req, http.StatusOK)

		csrf := GetCSRF(t, session, deletePath)

		// Prepare commit form
		commitForm := map[string]string{
			"_csrf":          csrf,
			"commit_summary": "Delete",
			"commit_message": "",
			"commit_choice":  "direct",
			"commit_mail_id": "-1",
		}

		// POST to delete the non-existent path - should return 200 with error message
		postReq := NewRequestWithValues(t, "POST", deletePath, commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusOK)

		// Verify that an error message is shown
		respBody := postResp.Body.String()
		assert.Contains(t, respBody, "no longer exists in this repository")

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

func TestRecursiveDeleteSpecialChars(t *testing.T) {
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
		deletePathViaUI(t, session, user2, repo1, treePathDirDel)

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
