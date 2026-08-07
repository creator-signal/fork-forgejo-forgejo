// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package git

import (
	"testing"

	"forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsRefProtected(t *testing.T) {
	testCases := []struct {
		name   string
		repoID int64

		ref            git.RefName
		expectedResult bool
	}{
		{
			name:           "Protected branch",
			repoID:         62,
			ref:            git.RefName("refs/heads/main"),
			expectedResult: true,
		},
		{
			name:           "Unprotected branch",
			repoID:         62,
			ref:            git.RefName("refs/heads/unprotected"),
			expectedResult: false,
		},
		{
			name:           "Protected tag",
			repoID:         62,
			ref:            git.RefName("refs/tags/v10"),
			expectedResult: true,
		},
		{
			name:           "Unprotected tag",
			repoID:         62,
			ref:            git.RefName("refs/tags/unprotected"),
			expectedResult: false,
		},
		{
			name:           "Empty ref",
			repoID:         62,
			ref:            git.RefName(""),
			expectedResult: false,
		},
		{
			name:           "Different repository",
			repoID:         10,
			ref:            git.RefName("refs/heads/main"),
			expectedResult: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require.NoError(t, unittest.PrepareTestDatabase())

			repo62 := unittest.AssertExistsAndLoadBean(t, &repo.Repository{ID: 62})
			unittest.AssertExistsAndLoadBean(t, &repo.Repository{ID: 10})

			protectedBranch := ProtectedBranch{ID: 144555, RepoID: repo62.ID, RuleName: "main"}
			unittest.AssertSuccessfulInsert(t, &protectedBranch)

			protectedTag := ProtectedTag{ID: 64882, RepoID: repo62.ID, NamePattern: "v*"}
			unittest.AssertSuccessfulInsert(t, &protectedTag)

			result, err := IsRefProtected(t.Context(), testCase.repoID, testCase.ref)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}
