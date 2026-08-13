// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/services/funding"
	"forgejo.org/tests/forgery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This is called from tests/integration/funding_retrievers_test.go
func FromDefaultBranch(t *testing.T) {
	// only a few basic cases, to demonstrate case insensitivity
	paths := []string{
		"funding.yml",
		// "Funding.yml", // TODO: enable these once the base case is working
		// "Funding.yaml",
		// ".github/FUNDING.yml",
		// ".forgejo/funding.yaml",
	}
	config := "custom: test.local\n"

	// This one works fine, probably because FilesInit uses CreateRepositoryDirectly instead of initRepo (where the error reportedly occurs):
	t.Run("empty repo", func(t *testing.T) {
		repo := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
			Files: forgery.FilesInit{},
		})

		f, err := funding.GetFundingFromDefaultBranch(t.Context(), repo)
		require.Nil(t, f)
		require.NotNil(t, err)
		assert.ErrorIs(t, err, funding.ErrFundingNotExist{})
	})

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			unittest.PrepareTestEnv(t)
			owner := forgery.CreateUser(t, nil)
			repo := forgery.CreateRepository(t, owner, &forgery.CreateRepositoryOptions{
				Files: forgery.MapFS{path: forgery.MapFile(config)}, // this should work, probably, but causes cryptic errors...
				// Files: forgery.FilesInit{},
			})
			// I've tried this instead of forgery.MapFS, to try to narrow the error down:
			// _, err := files_service.ChangeRepoFiles(t.Context(), repo, repo.Owner, &files_service.ChangeRepoFilesOptions{
			// 	Files: []*files_service.ChangeRepoFile{{
			// 		Operation:     "create",
			// 		TreePath:      path,
			// 		ContentReader: strings.NewReader(config),
			// 	}},
			// 	Message: "funding",
			// })
			// require.Nil(t, err)

			f, err := funding.GetFundingFromDefaultBranch(t.Context(), repo)
			require.Nil(t, err)
			require.NotNil(t, f)

			assert.Equal(t, path, f.ConfigPath)
			assert.Empty(t, f.Errors)
			assert.Len(t, f.Entries, 1)
			entry := f.Entries[0]
			assert.Equal(t, "custom", entry.ProviderName)
			assert.Equal(t, "test.local", entry.Title)
			assert.Equal(t, "test.local", entry.Value)
		})
	}
}
