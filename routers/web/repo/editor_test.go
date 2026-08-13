// Copyright 2018 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/services/contexttest"
	files_service "forgejo.org/services/repository/files"

	"github.com/stretchr/testify/assert"
)

func TestCleanUploadName(t *testing.T) {
	unittest.PrepareTestEnv(t)

	kases := map[string]string{
		".git/refs/master":               "",
		"/root/abc":                      "root/abc",
		"./../../abc":                    "abc",
		"a/../.git":                      "",
		"a/../../../abc":                 "abc",
		"../../../acd":                   "acd",
		"../../.git/abc":                 "",
		"..\\..\\.git/abc":               "..\\..\\.git/abc",
		"..\\../.git/abc":                "",
		"..\\../.git":                    "",
		"abc/../def":                     "def",
		".drone.yml":                     ".drone.yml",
		".abc/def/.drone.yml":            ".abc/def/.drone.yml",
		"..drone.yml.":                   "..drone.yml.",
		"..a.dotty...name...":            "..a.dotty...name...",
		"..a.dotty../.folder../.name...": "..a.dotty../.folder../.name...",
	}
	for k, v := range kases {
		assert.Equal(t, cleanUploadFileName(k), v)
	}
}

func TestGetUniquePatchBranchName(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx, _ := contexttest.MockContext(t, "user2/repo1")
	ctx.SetParams(":id", "1")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	expectedBranchName := "user2-patch-1"
	branchName := GetUniquePatchBranchName(ctx)
	assert.Equal(t, expectedBranchName, branchName)
}

func TestGetClosestParentWithFiles(t *testing.T) {
	unittest.PrepareTestEnv(t)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	branch := repo.DefaultBranch
	gitRepo, _ := gitrepo.OpenRepository(git.DefaultContext, repo)
	defer gitRepo.Close()
	commit, _ := gitRepo.GetBranchCommit(branch)
	var expectedTreePath string // Should return the root dir, empty string, since there are no subdirs in this repo
	for _, deletedFile := range []string{
		"dir1/dir2/dir3/file.txt",
		"file.txt",
	} {
		treePath := GetClosestParentWithFiles(deletedFile, commit)
		assert.Equal(t, expectedTreePath, treePath)
	}
}

func TestGetGitIdentityFromPatch(t *testing.T) {
	unittest.PrepareTestEnv(t)

	happyPathPatch := `From c862ec13473625a37edb761c7bc5c62f0917f067 Mon Sep 17 00:00:00 2001
From: test <test@test.com>
Date: Thu, 13 Aug 2026 15:56:28 +0200
Subject: [PATCH] fix a typo

---
 README.md | 2 +-
 1 file changed, 1 insertion(+), 1 deletion(-)

diff --git a/README.md b/README.md
index 0a63820..204b7f9 100644
--- a/README.md
+++ b/README.md
@@ -1 +1 @@
-# This is my awesom project!
+# This is my awesome project!
--
2.47.3
`
	gitIdentityHappyPath := getGitIdentityFromPatch(happyPathPatch)
	assert.Equal(t, &files_service.IdentityOptions{Name: "test", Email: "test@test.com"}, gitIdentityHappyPath)

	invalidPatchOneLine := "From c862ec13473625a37edb761c7bc5c62f0917f067 Mon Sep 17 00:00:00 2001"
	gitIdentityInvalidPatchOneLine := getGitIdentityFromPatch(invalidPatchOneLine)
	assert.Nil(t, gitIdentityInvalidPatchOneLine)

	invalidPatchSecondLineDoesntStartWithFrom := `From c862ec13473625a37edb761c7bc5c62f0917f067 Mon Sep 17 00:00:00 2001
test <test@test.com>`
	gitIdentityInvalidPatchSecondLineDoesntStartWithFrom := getGitIdentityFromPatch(invalidPatchSecondLineDoesntStartWithFrom)
	assert.Nil(t, gitIdentityInvalidPatchSecondLineDoesntStartWithFrom)

	invalidPatchNoName := `From c862ec13473625a37edb761c7bc5c62f0917f067 Mon Sep 17 00:00:00 2001
From: <test@test.com>`
	gitIdentityInvalidPatchNoName := getGitIdentityFromPatch(invalidPatchNoName)
	assert.Nil(t, gitIdentityInvalidPatchNoName)

	invalidPatchNoEmail := `From c862ec13473625a37edb761c7bc5c62f0917f067 Mon Sep 17 00:00:00 2001
From: test <>`
	gitIdentityInvalidPatchNoEmail := getGitIdentityFromPatch(invalidPatchNoEmail)
	assert.Nil(t, gitIdentityInvalidPatchNoEmail)
}
