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

	"github.com/stretchr/testify/assert"
)

func TestUIDeleteFormFile(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "dir2/dir3/file3.txt"
		expectedString := "File \"dir2/dir3/file3.txt\" has been deleted."

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		var lastCommitID string
		for i := range treePaths {
			fileResponse := createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			lastCommitID = fileResponse.Content.LastCommitSHA // Capture the last commit ID
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
			"last_commit":    lastCommitID,
		}

		// POST to delete the path
		postReq := NewRequestWithValues(t, "POST", deletePath, commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		// Follow redirect if present
		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			redirectResp := session.MakeRequest(t, verifyReq, http.StatusOK)
			doc := NewHTMLParser(t, redirectResp.Body)
			flashMessage := doc.Find("#flash-message.ui.positive.message").Text()
			assert.Contains(t, flashMessage, expectedString)
		} else {
			t.Fatalf("We should never be here. Something is broken outside the tested functionality.")
		}
	})
}

func TestUIDeleteFormFolder(t *testing.T) {
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		treePaths := []string{
			"file1.txt",
			"dir2/file2.txt",
			"dir2/dir3/file3.txt",
			"dir2/dir4/file4.txt",
		}

		treePathDirDel := "dir2/dir3"
		expectedString := "Folder \"dir2/dir3\" has been deleted."

		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})       // owner of the repo1
		repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}) // public repo

		// Get user2's token
		session := loginUser(t, user2.Name)
		token2 := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository, auth_model.AccessTokenScopeWriteUser)

		// Create test files
		var lastCommitID string
		for i := range treePaths {
			fileResponse := createFileWithAssertions(t, token2, user2, repo1, treePaths[i], "This is test text for: "+treePaths[i])
			lastCommitID = fileResponse.Content.LastCommitSHA // Capture the last commit ID
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
			"last_commit":    lastCommitID,
		}

		// POST to delete the path
		postReq := NewRequestWithValues(t, "POST", deletePath, commitForm)
		postResp := session.MakeRequest(t, postReq, http.StatusSeeOther)

		// Follow redirect if present
		redirectLocation := postResp.Header().Get("Location")
		if redirectLocation != "" {
			verifyReq := NewRequest(t, "GET", redirectLocation)
			redirectResp := session.MakeRequest(t, verifyReq, http.StatusOK)
			doc := NewHTMLParser(t, redirectResp.Body)
			flashMessage := doc.Find("#flash-message.ui.positive.message").Text()
			assert.Contains(t, flashMessage, expectedString)
		} else {
			t.Fatalf("We should never be here. Something is broken outside the tested functionality.")
		}
	})
}
