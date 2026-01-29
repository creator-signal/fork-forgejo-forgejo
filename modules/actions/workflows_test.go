// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsWorkflowsDetectMatched(t *testing.T) {
	testCases := []struct {
		desc           string
		commit         *git.Commit
		triggeredEvent webhook_module.HookEventType
		payload        api.Payloader
		yamlOn         string
		expected       bool
	}{
		{
			desc:           "HookEventCreate(create) matches GithubEventCreate(create)",
			triggeredEvent: webhook_module.HookEventCreate,
			payload:        nil,
			yamlOn:         "on: create",
			expected:       true,
		},
		{
			desc:           "HookEventIssues(issues) `opened` action matches GithubEventIssues(issues)",
			triggeredEvent: webhook_module.HookEventIssues,
			payload:        &api.IssuePayload{Action: api.HookIssueOpened},
			yamlOn:         "on: issues",
			expected:       true,
		},
		{
			desc:           "HookEventIssueComment(issue_comment) `created` action matches GithubEventIssueComment(issue_comment)",
			triggeredEvent: webhook_module.HookEventIssueComment,
			payload:        &api.IssueCommentPayload{Action: api.HookIssueCommentCreated},
			yamlOn:         "on:\n  issue_comment:\n    types: [created]",
			expected:       true,
		},

		{
			desc:           "HookEventIssues(issues) `milestoned` action matches GithubEventIssues(issues)",
			triggeredEvent: webhook_module.HookEventIssues,
			payload:        &api.IssuePayload{Action: api.HookIssueMilestoned},
			yamlOn:         "on: issues",
			expected:       true,
		},

		{
			desc:           "HookEventPullRequestSync(pull_request_sync) matches GithubEventPullRequest(pull_request)",
			triggeredEvent: webhook_module.HookEventPullRequestSync,
			payload:        &api.PullRequestPayload{Action: api.HookIssueSynchronized},
			yamlOn:         "on: pull_request",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `label_updated` action doesn't match GithubEventPullRequest(pull_request) with no activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueLabelUpdated},
			yamlOn:         "on: pull_request",
			expected:       false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `closed` action doesn't match GithubEventPullRequest(pull_request) with no activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueClosed},
			yamlOn:         "on: pull_request",
			expected:       false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `closed` action doesn't match GithubEventPullRequest(pull_request) with branches",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload: &api.PullRequestPayload{
				Action: api.HookIssueClosed,
				PullRequest: &api.PullRequest{
					Base: &api.PRBranchInfo{},
				},
			},
			yamlOn:   "on:\n  pull_request:\n    branches: [main]",
			expected: false,
		},
		{
			desc:           "HookEventPullRequest(pull_request) `label_updated` action matches GithubEventPullRequest(pull_request) with `label` activity type",
			triggeredEvent: webhook_module.HookEventPullRequest,
			payload:        &api.PullRequestPayload{Action: api.HookIssueLabelUpdated},
			yamlOn:         "on:\n  pull_request:\n    types: [labeled]",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequestReviewComment(pull_request_review_comment) matches GithubEventPullRequestReviewComment(pull_request_review_comment)",
			triggeredEvent: webhook_module.HookEventPullRequestReviewComment,
			payload:        &api.PullRequestPayload{Action: api.HookIssueReviewed},
			yamlOn:         "on:\n  pull_request_review_comment:\n    types: [created]",
			expected:       true,
		},
		{
			desc:           "HookEventPullRequestReviewRejected(pull_request_review_rejected) doesn't match GithubEventPullRequestReview(pull_request_review) with `dismissed` activity type (we don't support `dismissed` at present)",
			triggeredEvent: webhook_module.HookEventPullRequestReviewRejected,
			payload:        &api.PullRequestPayload{Action: api.HookIssueReviewed},
			yamlOn:         "on:\n  pull_request_review:\n    types: [dismissed]",
			expected:       false,
		},
		{
			desc:           "HookEventRelease(release) `published` action matches GithubEventRelease(release) with `published` activity type",
			triggeredEvent: webhook_module.HookEventRelease,
			payload:        &api.ReleasePayload{Action: api.HookReleasePublished},
			yamlOn:         "on:\n  release:\n    types: [published]",
			expected:       true,
		},
		{
			desc:           "HookEventRelease(updated) `updated` action matches GithubEventRelease(edited) with `edited` activity type",
			triggeredEvent: webhook_module.HookEventRelease,
			payload:        &api.ReleasePayload{Action: api.HookReleaseUpdated},
			yamlOn:         "on:\n  release:\n    types: [edited]",
			expected:       true,
		},

		{
			desc:           "HookEventPackage(package) `created` action doesn't match GithubEventRegistryPackage(registry_package) with `updated` activity type",
			triggeredEvent: webhook_module.HookEventPackage,
			payload:        &api.PackagePayload{Action: api.HookPackageCreated},
			yamlOn:         "on:\n  registry_package:\n    types: [updated]",
			expected:       false,
		},
		{
			desc:           "HookEventWiki(wiki) matches GithubEventGollum(gollum)",
			triggeredEvent: webhook_module.HookEventWiki,
			payload:        nil,
			yamlOn:         "on: gollum",
			expected:       true,
		},
		{
			desc:           "HookEventSchedule(schedule) matches GithubEventSchedule(schedule)",
			triggeredEvent: webhook_module.HookEventSchedule,
			payload:        nil,
			yamlOn:         "on: schedule",
			expected:       true,
		},
		{
			desc:           "HookEventWorkflowDispatch(workflow_dispatch) matches GithubEventWorkflowDispatch(workflow_dispatch)",
			triggeredEvent: webhook_module.HookEventWorkflowDispatch,
			payload:        nil,
			yamlOn:         "on: workflow_dispatch",
			expected:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			evts, err := GetEventsFromContent([]byte(tc.yamlOn))
			require.NoError(t, err)
			assert.Len(t, evts, 1)
			assert.Equal(t, tc.expected, detectMatched(nil, tc.commit, tc.triggeredEvent, tc.payload, evts[0]))
		})
	}
}

func TestMatchPushEvent(t *testing.T) {
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(t.Context()))

	committer := git.Signature{
		Email: "test@example.com",
		Name:  "Test",
		When:  time.Now(),
	}

	repoHome := t.TempDir()

	// create initial commit with README.md
	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, "src"), os.ModePerm))
	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, "docs"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, "README.md"), []byte("# Test"), 0o644))
	require.NoError(t, git.InitRepository(t.Context(), repoHome, false, git.Sha1ObjectFormat.Name()))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Initial commit", Committer: &committer}))

	gitRepo, err := git.OpenRepository(t.Context(), repoHome)
	require.NoError(t, err)
	defer gitRepo.Close()

	headBranch, err := gitRepo.GetHEADBranch()
	require.NoError(t, err)

	initialCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)

	// create commit with src/file.go change
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, "src", "file.go"), []byte("package main"), 0o644))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Add src file", Committer: &committer}))

	srcCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)
	srcCommit, err := gitRepo.GetCommit(srcCommitID)
	require.NoError(t, err)

	// create commit with docs/file.md change
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, "docs", "file.md"), []byte("# Docs"), 0o644))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Add docs file", Committer: &committer}))

	docCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)
	docCommit, err := gitRepo.GetCommit(docCommitID)
	require.NoError(t, err)

	commits := map[string]string{
		"initial": initialCommitID,
		"src":     srcCommitID,
		"doc":     docCommitID,
	}
	commitObjs := map[string]*git.Commit{
		"src": srcCommit,
		"doc": docCommit,
	}

	testCases := []struct {
		desc           string
		yamlOn         string
		ref            string
		previousCommit string
		currentCommit  string
		expected       bool
	}{
		// no filter
		{
			desc:     "no filter matches any push",
			yamlOn:   "on: push",
			ref:      "refs/heads/main",
			expected: true,
		},

		// branches filter
		{
			desc:     "branches filter matches branch push",
			yamlOn:   "on:\n  push:\n    branches: [main]",
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			desc:     "branches filter does not match different branch",
			yamlOn:   "on:\n  push:\n    branches: [main]",
			ref:      "refs/heads/develop",
			expected: false,
		},
		{
			desc:     "branches filter does not match tag push",
			yamlOn:   "on:\n  push:\n    branches: [main]",
			ref:      "refs/tags/v1.0",
			expected: false,
		},

		// branches-ignore filter
		{
			desc:     "branches-ignore filter matches branch not in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]",
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			desc:     "branches-ignore filter does not match branch in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]",
			ref:      "refs/heads/main",
			expected: false,
		},
		{
			desc:     "branches-ignore filter does not match tag push",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]",
			ref:      "refs/tags/v1.0",
			expected: false,
		},

		// tags filter
		{
			desc:     "tags filter matches tag push",
			yamlOn:   "on:\n  push:\n    tags: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: true,
		},
		{
			desc:     "tags filter does not match different tag",
			yamlOn:   "on:\n  push:\n    tags: ['v*']",
			ref:      "refs/tags/release",
			expected: false,
		},
		{
			desc:     "tags filter does not match branch push",
			yamlOn:   "on:\n  push:\n    tags: ['v*']",
			ref:      "refs/heads/main",
			expected: false,
		},

		// tags-ignore filter
		{
			desc:     "tags-ignore filter matches tag not in ignore list",
			yamlOn:   "on:\n  push:\n    tags-ignore: ['v*']",
			ref:      "refs/tags/release",
			expected: true,
		},
		{
			desc:     "tags-ignore filter does not match tag in ignore list",
			yamlOn:   "on:\n  push:\n    tags-ignore: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: false,
		},
		{
			desc:     "tags-ignore filter does not match branch push",
			yamlOn:   "on:\n  push:\n    tags-ignore: ['v*']",
			ref:      "refs/heads/main",
			expected: false,
		},

		// paths filter
		{
			desc:           "paths filter matches when path matches",
			yamlOn:         "on:\n  push:\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       true,
		},
		{
			desc:           "paths filter does not match when path does not match",
			yamlOn:         "on:\n  push:\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "src",
			currentCommit:  "doc",
			expected:       false,
		},
		{
			desc:     "paths filter is ignored for tag push",
			yamlOn:   "on:\n  push:\n    paths:\n      - src/**",
			ref:      "refs/tags/v1.0",
			expected: true,
		},

		// paths-ignore filter
		{
			desc:           "paths-ignore filter matches when path not in ignore list",
			yamlOn:         "on:\n  push:\n    paths-ignore:\n      - docs/**",
			ref:            "refs/heads/main",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       true,
		},
		{
			desc:           "paths-ignore filter does not match when path in ignore list",
			yamlOn:         "on:\n  push:\n    paths-ignore:\n      - docs/**",
			ref:            "refs/heads/main",
			previousCommit: "src",
			currentCommit:  "doc",
			expected:       false,
		},

		// branches + tags (OR logic)
		{
			desc:     "branches and tags filter matches branch",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags: ['v*']",
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			desc:     "branches and tags filter matches tag",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: true,
		},
		{
			desc:     "branches and tags filter does not match wrong branch",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags: ['v*']",
			ref:      "refs/heads/develop",
			expected: false,
		},

		// branches + paths (AND logic)
		{
			desc:           "branches and paths filter matches when both match",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       true,
		},
		{
			desc:           "branches and paths filter does not match when only branch matches",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "src",
			currentCommit:  "doc",
			expected:       false,
		},
		{
			desc:           "branches and paths filter does not match when only path matches",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    paths:\n      - src/**",
			ref:            "refs/heads/develop",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       false,
		},

		// tags + paths (paths ignored for tags)
		{
			desc:     "tags and paths filter matches tag (paths ignored)",
			yamlOn:   "on:\n  push:\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:      "refs/tags/v1.0",
			expected: true,
		},

		// branches + tags + paths
		{
			desc:           "branches, tags and paths filter matches branch with path",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       true,
		},
		{
			desc:     "branches, tags and paths filter matches tag (paths ignored)",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:      "refs/tags/v1.0",
			expected: true,
		},
		{
			desc:           "branches, tags and paths filter does not match when path does not match",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:            "refs/heads/main",
			previousCommit: "src",
			currentCommit:  "doc",
			expected:       false,
		},
		{
			desc:           "branches, tags and paths filter does not match when branch does not match",
			yamlOn:         "on:\n  push:\n    branches: [main]\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:            "refs/heads/develop",
			previousCommit: "initial",
			currentCommit:  "src",
			expected:       false,
		},
		{
			desc:     "branches, tags and paths filter does not match when tag does not match",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags: ['v*']\n    paths:\n      - src/**",
			ref:      "refs/tags/release",
			expected: false,
		},

		// branches-ignore + tags (OR logic)
		{
			desc:     "branches-ignore and tags filter matches branch not in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags: ['v*']",
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			desc:     "branches-ignore and tags filter does not match branch in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags: ['v*']",
			ref:      "refs/heads/main",
			expected: false,
		},
		{
			desc:     "branches-ignore and tags filter matches tag",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: true,
		},
		{
			desc:     "branches-ignore and tags filter does not match wrong tag",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags: ['v*']",
			ref:      "refs/tags/release",
			expected: false,
		},

		// branches + tags-ignore (OR logic)
		{
			desc:     "branches and tags-ignore filter matches branch",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/heads/main",
			expected: true,
		},
		{
			desc:     "branches and tags-ignore filter does not match wrong branch",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/heads/develop",
			expected: false,
		},
		{
			desc:     "branches and tags-ignore filter matches tag not in ignore list",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/tags/release",
			expected: true,
		},
		{
			desc:     "branches and tags-ignore filter does not match tag in ignore list",
			yamlOn:   "on:\n  push:\n    branches: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: false,
		},

		// branches-ignore + tags-ignore (OR logic)
		{
			desc:     "branches-ignore and tags-ignore filter matches branch not in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/heads/develop",
			expected: true,
		},
		{
			desc:     "branches-ignore and tags-ignore filter does not match branch in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/heads/main",
			expected: false,
		},
		{
			desc:     "branches-ignore and tags-ignore filter matches tag not in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/tags/release",
			expected: true,
		},
		{
			desc:     "branches-ignore and tags-ignore filter does not match tag in ignore list",
			yamlOn:   "on:\n  push:\n    branches-ignore: [main]\n    tags-ignore: ['v*']",
			ref:      "refs/tags/v1.0",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			evts, err := GetEventsFromContent([]byte(tc.yamlOn))
			require.NoError(t, err)
			require.Len(t, evts, 1)

			var commit *git.Commit
			var previousCommitID string

			if tc.currentCommit != "" {
				commit = commitObjs[tc.currentCommit]
			}
			if tc.previousCommit != "" {
				previousCommitID = commits[tc.previousCommit]
			}

			payload := &api.PushPayload{
				Ref:    tc.ref,
				Before: previousCommitID,
			}

			assert.Equal(t, tc.expected, detectMatched(gitRepo, commit, webhook_module.HookEventPush, payload, evts[0]))
		})
	}
}

func TestActionsWorkflowsListWorkflowsReturnsNoWorkflowsIfThereAreNone(t *testing.T) {
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(t.Context()))

	committer := git.Signature{
		Email: "jane@example.com",
		Name:  "Jane",
		When:  time.Now(),
	}
	repoHome := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(repoHome, "README.md"), []byte("My project"), 0o644))

	require.NoError(t, git.InitRepository(t.Context(), repoHome, false, git.Sha1ObjectFormat.Name()))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Import", Committer: &committer}))

	gitRepo, err := git.OpenRepository(t.Context(), repoHome)
	require.NoError(t, err)
	defer gitRepo.Close()

	headBranch, err := gitRepo.GetHEADBranch()
	require.NoError(t, err)

	lastCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)

	lastCommit, err := gitRepo.GetCommit(lastCommitID)
	require.NoError(t, err)

	source, workflows, err := ListWorkflows(lastCommit)
	require.NoError(t, err)

	assert.Empty(t, source)
	assert.Empty(t, workflows)
}

func TestActionsWorkflowsListWorkflowsIgnoresNonWorkflowFiles(t *testing.T) {
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(t.Context()))

	committer := git.Signature{
		Email: "jane@example.com",
		Name:  "Jane",
		When:  time.Now(),
	}
	githubWorkflow := []byte(`
name: GitHub Workflow
on:
  push:
jobs:
  do-something:
    runs-on: ubuntu-latest
    steps:
      - run: echo 'Hello GitHub'
`)
	repoHome := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, ".forgejo/workflows"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".forgejo/workflows", "README.md"), []byte("My project"), 0o644))

	// Prepare a valid workflow in .github/workflows to verify that it is ignored because .forgejo/workflows is present.
	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, ".github/workflows"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".github/workflows", "github.yaml"), githubWorkflow, 0o644))

	require.NoError(t, git.InitRepository(t.Context(), repoHome, false, git.Sha1ObjectFormat.Name()))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Import", Committer: &committer}))

	gitRepo, err := git.OpenRepository(t.Context(), repoHome)
	require.NoError(t, err)
	defer gitRepo.Close()

	headBranch, err := gitRepo.GetHEADBranch()
	require.NoError(t, err)

	lastCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)

	lastCommit, err := gitRepo.GetCommit(lastCommitID)
	require.NoError(t, err)

	source, workflows, err := ListWorkflows(lastCommit)
	require.NoError(t, err)

	assert.Equal(t, ".forgejo/workflows", source)
	assert.Empty(t, workflows)
}

func TestActionsWorkflowsListWorkflowsReturnsForgejoWorkflowsOnly(t *testing.T) {
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(t.Context()))

	committer := git.Signature{
		Email: "jane@example.com",
		Name:  "Jane",
		When:  time.Now(),
	}
	forgejoWorkflow := []byte(`
name: Forgejo Workflow
on:
  push:
jobs:
  do-something:
    runs-on: ubuntu-latest
    steps:
      - run: echo 'Hello Forgejo'
`)
	githubWorkflow := []byte(`
name: GitHub Workflow
on:
  push:
jobs:
  do-something:
    runs-on: ubuntu-latest
    steps:
      - run: echo 'Hello GitHub'
`)
	repoHome := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, ".forgejo/workflows"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".forgejo/workflows", "forgejo.yaml"), forgejoWorkflow, 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, ".github/workflows"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".github/workflows", "github.yaml"), githubWorkflow, 0o644))

	require.NoError(t, git.InitRepository(t.Context(), repoHome, false, git.Sha1ObjectFormat.Name()))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Import", Committer: &committer}))

	gitRepo, err := git.OpenRepository(t.Context(), repoHome)
	require.NoError(t, err)
	defer gitRepo.Close()

	headBranch, err := gitRepo.GetHEADBranch()
	require.NoError(t, err)

	lastCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)

	lastCommit, err := gitRepo.GetCommit(lastCommitID)
	require.NoError(t, err)

	source, workflows, err := ListWorkflows(lastCommit)
	require.NoError(t, err)

	assert.Len(t, workflows, 1)
	assert.Equal(t, ".forgejo/workflows", source)
	assert.Equal(t, "forgejo.yaml", workflows[0].Name())
}

func TestActionsWorkflowsListWorkflowsReturnsGitHubWorkflowsIfForgejoWorkflowsAbsent(t *testing.T) {
	t.Cleanup(test.MockVariableValue(&setting.Git.HomePath, t.TempDir()))
	require.NoError(t, git.InitSimple(t.Context()))

	committer := git.Signature{
		Email: "jane@example.com",
		Name:  "Jane",
		When:  time.Now(),
	}
	buildWorkflow := []byte(`
name: Build
on:
  push:
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - run: echo 'We are building'
`)
	testWorkflow := []byte(`
name: Test
on:
  push:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo 'We are testing'
`)
	repoHome := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoHome, ".github/workflows"), os.ModePerm))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".github/workflows", "build.yaml"), buildWorkflow, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repoHome, ".github/workflows", "test.yml"), testWorkflow, 0o644))

	require.NoError(t, git.InitRepository(t.Context(), repoHome, false, git.Sha1ObjectFormat.Name()))
	require.NoError(t, git.AddChanges(repoHome, true))
	require.NoError(t, git.CommitChanges(repoHome, git.CommitChangesOptions{Message: "Import", Committer: &committer}))

	gitRepo, err := git.OpenRepository(t.Context(), repoHome)
	require.NoError(t, err)
	defer gitRepo.Close()

	headBranch, err := gitRepo.GetHEADBranch()
	require.NoError(t, err)

	lastCommitID, err := gitRepo.GetBranchCommitID(headBranch.Name)
	require.NoError(t, err)

	lastCommit, err := gitRepo.GetCommit(lastCommitID)
	require.NoError(t, err)

	source, workflows, err := ListWorkflows(lastCommit)
	require.NoError(t, err)

	assert.Len(t, workflows, 2)
	assert.Equal(t, ".github/workflows", source)
	assert.Equal(t, "build.yaml", workflows[0].Name())
	assert.Equal(t, "test.yml", workflows[1].Name())
}
