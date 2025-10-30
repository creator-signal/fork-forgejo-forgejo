// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"fmt"
	"testing"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRunBefore(t *testing.T) {
}

func TestSetConcurrencyGroup(t *testing.T) {
	run := ActionRun{}
	run.SetConcurrencyGroup("abc123")
	assert.Equal(t, "abc123", run.ConcurrencyGroup)
	run.SetConcurrencyGroup("ABC123") // case should collapse in SetConcurrencyGroup
	assert.Equal(t, "abc123", run.ConcurrencyGroup)
}

func TestSetDefaultConcurrencyGroup(t *testing.T) {
	run := ActionRun{
		Ref:          "refs/heads/main",
		WorkflowID:   "testing",
		TriggerEvent: "pull_request",
	}
	run.SetDefaultConcurrencyGroup()
	assert.Equal(t, "refs/heads/main_testing_pull_request__auto", run.ConcurrencyGroup)
	run = ActionRun{
		Ref:          "refs/heads/main",
		WorkflowID:   "TESTING", // case should collapse in SetDefaultConcurrencyGroup
		TriggerEvent: "pull_request",
	}
	run.SetDefaultConcurrencyGroup()
	assert.Equal(t, "refs/heads/main_testing_pull_request__auto", run.ConcurrencyGroup)
}

func TestUpdateRepoRunsNumbers(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("Normal", func(t *testing.T) {
		t.Run("Repo 1", func(t *testing.T) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

			require.NoError(t, updateRepoRunsNumbers(t.Context(), repo))

			repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
			assert.Equal(t, 1, repo.NumActionRuns)
			assert.Equal(t, 1, repo.NumClosedActionRuns)
		})

		t.Run("Repo 4", func(t *testing.T) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})

			require.NoError(t, updateRepoRunsNumbers(t.Context(), repo))

			repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4})
			assert.Equal(t, 4, repo.NumActionRuns)
			assert.Equal(t, 4, repo.NumClosedActionRuns)
		})

		t.Run("Repo 63", func(t *testing.T) {
			repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 63})

			require.NoError(t, updateRepoRunsNumbers(t.Context(), repo))

			repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 63})
			assert.Equal(t, 3, repo.NumActionRuns)
			assert.Equal(t, 2, repo.NumClosedActionRuns)
		})
	})

	t.Run("Columns specifc", func(t *testing.T) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		repo.Name = "ishouldnotbeupdated"

		require.NoError(t, updateRepoRunsNumbers(t.Context(), repo))

		repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
		assert.Equal(t, "repo1", repo.Name)
	})
}

func TestActionRun_GetRunsNotDoneByRepoIDAndPullRequestPosterID(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repoID := int64(10)
	pullRequestID := int64(3)
	pullRequestPosterID := int64(30)

	runDone := &ActionRun{
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: pullRequestPosterID,
		Status:              StatusSuccess,
	}
	require.NoError(t, InsertRun(t.Context(), runDone, nil))

	unrelatedUser := int64(5)
	runNotByPoster := &ActionRun{
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: unrelatedUser,
		Status:              StatusRunning,
	}
	require.NoError(t, InsertRun(t.Context(), runNotByPoster, nil))

	unrelatedRepository := int64(6)
	runNotInTheSameRepository := &ActionRun{
		RepoID:              unrelatedRepository,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: pullRequestPosterID,
		Status:              StatusSuccess,
	}
	require.NoError(t, InsertRun(t.Context(), runNotInTheSameRepository, nil))

	for _, status := range []Status{StatusUnknown, StatusWaiting, StatusRunning} {
		t.Run(fmt.Sprintf("%s", status), func(t *testing.T) {
			runNotDone := &ActionRun{
				RepoID:              repoID,
				PullRequestID:       pullRequestID,
				Status:              status,
				PullRequestPosterID: pullRequestPosterID,
			}
			require.NoError(t, InsertRun(t.Context(), runNotDone, nil))
			runs, err := GetRunsNotDoneByRepoIDAndPullRequestPosterID(t.Context(), repoID, pullRequestPosterID)
			require.NoError(t, err)
			require.Len(t, runs, 1)
			run := runs[0]
			assert.Equal(t, runNotDone.ID, run.ID)
			assert.Equal(t, runNotDone.Status, run.Status)
			unittest.AssertSuccessfulDelete(t, run)
		})
	}
}

func TestActionRun_NeedApproval(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pullRequestPosterID := int64(4)
	repoID := int64(10)
	pullRequestID := int64(2)
	runDoesNotNeedApproval := &ActionRun{
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: pullRequestPosterID,
	}
	require.NoError(t, InsertRun(t.Context(), runDoesNotNeedApproval, nil))
	unrelatedRepository := int64(6)
	runNotInTheSameRepository := &ActionRun{
		RepoID:              unrelatedRepository,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: pullRequestPosterID,
		NeedApproval:        true,
	}
	require.NoError(t, InsertRun(t.Context(), runNotInTheSameRepository, nil))
	unrelatedPullRequest := int64(3)
	runNotInTheSamePullRequest := &ActionRun{
		RepoID:              repoID,
		PullRequestID:       unrelatedPullRequest,
		PullRequestPosterID: pullRequestPosterID,
		NeedApproval:        true,
	}
	require.NoError(t, InsertRun(t.Context(), runNotInTheSamePullRequest, nil))

	t.Run("HasRunThatNeedApproval is false", func(t *testing.T) {
		has, err := HasRunThatNeedApproval(t.Context(), repoID, pullRequestID)
		require.NoError(t, err)
		assert.False(t, has)
	})

	runNeedApproval := &ActionRun{
		RepoID:              repoID,
		PullRequestID:       pullRequestID,
		PullRequestPosterID: pullRequestPosterID,
		NeedApproval:        true,
	}
	require.NoError(t, InsertRun(t.Context(), runNeedApproval, nil))

	t.Run("HasRunThatNeedApproval is true", func(t *testing.T) {
		has, err := HasRunThatNeedApproval(t.Context(), repoID, pullRequestID)
		require.NoError(t, err)
		assert.True(t, has)
	})

	assertApprovalEqual := func(t *testing.T, expected, actual *ActionRun) {
		t.Helper()
		assert.Equal(t, expected.RepoID, actual.RepoID)
		assert.Equal(t, expected.PullRequestID, actual.PullRequestID)
		assert.Equal(t, expected.PullRequestPosterID, actual.PullRequestPosterID)
		assert.Equal(t, expected.NeedApproval, actual.NeedApproval)
	}

	t.Run("GetRunsThatNeedApproval", func(t *testing.T) {
		runs, err := GetRunsThatNeedApproval(t.Context(), repoID, pullRequestID)
		require.NoError(t, err)
		require.Len(t, runs, 1)
		assertApprovalEqual(t, runNeedApproval, runs[0])
	})
}
