// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"

	"code.forgejo.org/forgejo/runner/v12/act/jobparser"
	"code.forgejo.org/forgejo/runner/v12/act/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testWorkflow string = `on:
  workflow_call:
    inputs:
      example-string-required:
        required: true
        type: string

name: test
jobs:
  job1:
    name: "job1 (local)"
    runs-on: ubuntu-slim
    steps:
      - name: Echo inputs
        run: |
          echo example-string-required="${{ inputs.example-string-required }}"

`

func TestExpandLocalReusableWorkflows(t *testing.T) {
	gitRepo, err := git.OpenRepository(git.DefaultContext, "./TestExpandLocalReusableWorkflows")
	require.NoError(t, err)
	defer gitRepo.Close()

	commit, err := gitRepo.GetCommit("e3868ecb4f8b483fc0bdd422561bf0062a7df907")
	require.NoError(t, err)

	fetcher := expandLocalReusableWorkflows(commit)
	require.NotNil(t, fetcher)

	content, err := fetcher("./.forgejo/workflows/reusable-1.yml")
	require.NoError(t, err)
	assert.Equal(t, testWorkflow, string(content))

	_, err = fetcher("./forgejo/workflows/reusable-2.yml")
	assert.ErrorContains(t, err, "expanding reusable workflow failed to access path ./forgejo/workflows/reusable-2.yml: object does not exist")
}

func replaceTestRepo(t *testing.T, owner, repo, replacement string) {
	t.Helper()

	// Copy the repository into the target path that `gitrepo.OpenRepository` will look for it.
	repoPath := filepath.Join(setting.RepoRootPath, strings.ToLower(owner), strings.ToLower(repo)+".git")
	err := os.RemoveAll(repoPath) // there's a default repo copied here by the fixture setup that we want to replace
	require.NoError(t, err)
	err = os.CopyFS(repoPath, os.DirFS(replacement))
	require.NoError(t, err)
}

func TestLazyRepoExpandLocalReusableWorkflows(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// Shouldn't need valid content if we never call the lazy evaluator
	lazy1, cleanup := lazyRepoExpandLocalReusableWorkflow(t.Context(), -123456, "this is not a valid commit SHA")
	assert.NotNil(t, lazy1)
	assert.NotNil(t, cleanup)
	cleanup()

	replaceTestRepo(t, "user2", "repo1", "./TestExpandLocalReusableWorkflows")

	lazy2, cleanup := lazyRepoExpandLocalReusableWorkflow(t.Context(), 1, "e3868ecb4f8b483fc0bdd422561bf0062a7df907")
	assert.NotNil(t, lazy2)
	assert.NotNil(t, cleanup)
	content, err := lazy2("./.forgejo/workflows/reusable-1.yml")
	require.NoError(t, err)
	assert.Equal(t, testWorkflow, string(content))
	cleanup()
}

func TestExpandRemoteReusableWorkflows(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	baseURL := "https://github.com"
	tests := []struct {
		name          string
		ref           *model.RemoteReusableWorkflowWithBaseURL
		errIs         error
		errorContains string
		repo          string
	}{
		{
			name: "baseURL",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				BaseURL:                &baseURL,
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{},
			},
			errIs: jobparser.ErrUnsupportedReusableWorkflowFetch,
		},
		{
			name: "non-existent owner",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{
					Org: "owner-does-not-exist",
				},
			},
			errorContains: "owner-does-not-exist: user does not exist",
		},
		{
			name: "non-public owner",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{
					Org: "user33",
				},
			},
			errorContains: "user33: user does not exist",
		},
		{
			name: "non-existent repo",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{
					Org:  "user2",
					Repo: "repo10000",
				},
			},
			errorContains: "repo10000: repo does not exist",
		},
		{
			name: "non-public repo",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{
					Org:  "user2",
					Repo: "repo2",
				},
			},
			errorContains: "repo2: repo does not exist",
		},
		{
			name: "public repo",
			ref: &model.RemoteReusableWorkflowWithBaseURL{
				RemoteReusableWorkflow: model.RemoteReusableWorkflow{
					Org:         "user2",
					Repo:        "repo1",
					GitPlatform: "forgejo",
					Filename:    "reusable-1.yml",
					Ref:         "main",
				},
			},
			repo: "./TestExpandLocalReusableWorkflows",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.repo != "" {
				replaceTestRepo(t, tt.ref.Org, tt.ref.Repo, tt.repo)
			}

			fetcher := expandRemoteReusableWorkflows(t.Context())
			content, err := fetcher(tt.ref)
			if tt.errIs != nil {
				require.ErrorIs(t, err, tt.errIs)
			} else if tt.errorContains != "" {
				require.ErrorContains(t, err, tt.errorContains)
			} else {
				require.NoError(t, err)
				assert.Equal(t, testWorkflow, string(content))
			}
		})
	}
}
