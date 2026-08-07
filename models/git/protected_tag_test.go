// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsUserAllowed(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pt := &ProtectedTag{}
	allowed, err := IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.False(t, allowed)

	pt = &ProtectedTag{
		AllowlistUserIDs: []int64{1},
	}
	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.False(t, allowed)

	pt = &ProtectedTag{
		AllowlistTeamIDs: []int64{1},
	}
	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.False(t, allowed)

	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.True(t, allowed)

	pt = &ProtectedTag{
		AllowlistUserIDs: []int64{1},
		AllowlistTeamIDs: []int64{1},
	}
	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 1)
	require.NoError(t, err)
	assert.True(t, allowed)

	allowed, err = IsUserAllowedModifyTag(db.DefaultContext, pt, 2)
	require.NoError(t, err)
	assert.True(t, allowed)
}

func TestIsUserAllowedToControlTag(t *testing.T) {
	cases := []struct {
		name    string
		userid  int64
		allowed bool
	}{
		{
			name:    "test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "test-gitea",
			userid:  1,
			allowed: true,
		},
		{
			name:    "test-gitea",
			userid:  3,
			allowed: false,
		},
		{
			name:    "gitea-test",
			userid:  1,
			allowed: true,
		},
		{
			name:    "gitea-test",
			userid:  3,
			allowed: true,
		},
		{
			name:    "v-1",
			userid:  1,
			allowed: false,
		},
		{
			name:    "v-1",
			userid:  2,
			allowed: true,
		},
		{
			name:    "release",
			userid:  1,
			allowed: false,
		},
	}

	t.Run("Glob", func(t *testing.T) {
		protectedTags := []*ProtectedTag{
			{
				NamePattern:      `*gitea`,
				AllowlistUserIDs: []int64{1},
			},
			{
				NamePattern:      `v-*`,
				AllowlistUserIDs: []int64{2},
			},
			{
				NamePattern: "release",
			},
		}

		for n, c := range cases {
			isAllowed, err := IsUserAllowedToControlTag(db.DefaultContext, protectedTags, c.name, c.userid)
			require.NoError(t, err)
			assert.Equal(t, c.allowed, isAllowed, "case %d: error should match", n)
		}
	})

	t.Run("Regex", func(t *testing.T) {
		protectedTags := []*ProtectedTag{
			{
				NamePattern:      `/gitea\z/`,
				AllowlistUserIDs: []int64{1},
			},
			{
				NamePattern:      `/\Av-/`,
				AllowlistUserIDs: []int64{2},
			},
			{
				NamePattern: "/release/",
			},
		}

		for n, c := range cases {
			isAllowed, err := IsUserAllowedToControlTag(db.DefaultContext, protectedTags, c.name, c.userid)
			require.NoError(t, err)
			assert.Equal(t, c.allowed, isAllowed, "case %d: error should match", n)
		}
	})
}

func TestProtectedTag_Affects(t *testing.T) {
	testCases := []struct {
		name           string
		protectedTag   *ProtectedTag
		ref            git.RefName
		expectedResult bool
		expectedError  string
	}{
		{
			name:           "Commit",
			protectedTag:   &ProtectedTag{GlobPattern: glob.MustCompile("v*")},
			ref:            git.RefName("e83e55179e8a4b697f26e2d786caafeef464a488"),
			expectedResult: false,
		},
		{
			name:           "Branch",
			protectedTag:   &ProtectedTag{GlobPattern: glob.MustCompile("v*")},
			ref:            git.RefName("refs/heads/versions"),
			expectedResult: false,
		},
		{
			name:           "Matching glob",
			protectedTag:   &ProtectedTag{NamePattern: "v*"},
			ref:            git.RefName("refs/tags/v1"),
			expectedResult: true,
		},
		{
			name:           "Matching regular expression",
			protectedTag:   &ProtectedTag{NamePattern: "/^v[0-9]$/"},
			ref:            git.RefName("refs/tags/v1"),
			expectedResult: true,
		},
		{
			name:           "Other tag",
			protectedTag:   &ProtectedTag{NamePattern: "v*"},
			ref:            git.RefName("refs/tags/1v"),
			expectedResult: false,
		},
		{
			name:           "Empty ref",
			protectedTag:   &ProtectedTag{NamePattern: "v*"},
			ref:            git.RefName(""),
			expectedResult: false,
		},
		{
			name:          "InvalidPattern",
			protectedTag:  &ProtectedTag{NamePattern: "/(v*/"},
			ref:           git.RefName("refs/tags/1v"),
			expectedError: "error parsing regexp",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := testCase.protectedTag.Affects(testCase.ref)
			if testCase.expectedError != "" {
				require.ErrorContains(t, err, testCase.expectedError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.expectedResult, result)
		})
	}
}
