// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	actions_module "forgejo.org/modules/actions"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsTrust_ChangeStatus(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	repoID := int64(10)
	pullRequestPosterID := int64(30)

	runDone := &actions_model.ActionRun{
		RepoID:              repoID,
		PullRequestPosterID: pullRequestPosterID,
		Status:              actions_model.StatusSuccess,
	}
	require.NoError(t, actions_model.InsertRun(t.Context(), runDone, nil))

	runNotByPoster := &actions_model.ActionRun{
		RepoID:              repoID,
		PullRequestPosterID: 43243,
		Status:              actions_model.StatusRunning,
	}
	require.NoError(t, actions_model.InsertRun(t.Context(), runNotByPoster, nil))

	runNotInTheSameRepository := &actions_model.ActionRun{
		RepoID:              5,
		PullRequestPosterID: pullRequestPosterID,
		Status:              actions_model.StatusSuccess,
	}
	require.NoError(t, actions_model.InsertRun(t.Context(), runNotInTheSameRepository, nil))

	t.Run("RevokeTrustByRepoIDAndPosterID", func(t *testing.T) {
		singleWorkflows, err := actions_module.JobParser([]byte(`
jobs:
  job:
    runs-on: docker
    steps:
      - run: echo OK
`))
		require.NoError(t, err)
		require.Len(t, singleWorkflows, 1)
		runNotDone := &actions_model.ActionRun{
			TriggerUserID:       2,
			RepoID:              repoID,
			Status:              actions_model.StatusWaiting,
			PullRequestPosterID: pullRequestPosterID,
		}
		require.NoError(t, actions_model.InsertRun(t.Context(), runNotDone, singleWorkflows))
		require.NoError(t, actions_model.InsertActionUser(t.Context(), &actions_model.ActionUser{
			UserID:                  pullRequestPosterID,
			RepoID:                  repoID,
			TrustedWithPullRequests: true,
		}))

		previousCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), pullRequestPosterID, repoID)
		require.NoError(t, err)

		require.NoError(t, RevokeTrustByRepoIDAndPosterID(t.Context(), repoID, pullRequestPosterID))

		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), pullRequestPosterID, repoID)
		assert.True(t, actions_model.IsErrUserNotExist(err))
		currentCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})
		assert.Equal(t, previousCancelledCount+1, currentCancelledCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotDone.ID})
		assert.Equal(t, actions_model.StatusCancelled.String(), run.Status.String())
	})

	createPullRequestRun := func(t *testing.T, pullRequestID, repoID int64) *actions_model.ActionRun {
		t.Helper()
		singleWorkflows, err := actions_module.JobParser([]byte(`
jobs:
  job:
    runs-on: docker
    steps:
      - run: echo OK
`))
		require.NoError(t, err)
		require.Len(t, singleWorkflows, 1)
		runNotApproved := &actions_model.ActionRun{
			TriggerUserID:       2,
			RepoID:              repoID,
			Status:              actions_model.StatusWaiting,
			NeedApproval:        true,
			PullRequestID:       pullRequestID,
			PullRequestPosterID: pullRequestPosterID,
		}
		require.NoError(t, actions_model.InsertRun(t.Context(), runNotApproved, singleWorkflows))
		return runNotApproved
	}

	t.Run("PullRequestCancel", func(t *testing.T) {
		pullRequestID := int64(485)
		runNotApproved := createPullRequestRun(t, pullRequestID, repoID)

		previousCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})

		require.NoError(t, PullRequestCancel(t.Context(), &issues_model.PullRequest{
			ID: pullRequestID,
			Issue: &issues_model.Issue{
				RepoID: repoID,
			},
		}))

		currentCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})
		assert.Equal(t, previousCancelledCount+1, currentCancelledCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusCancelled.String(), run.Status.String())
	})

	t.Run("UpdateTrustedWithPullRequest not trusted", func(t *testing.T) {
		pullRequestID := int64(485)
		runNotApproved := createPullRequestRun(t, pullRequestID, repoID)

		previousCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})

		require.NoError(t, UpdateTrustedWithPullRequest(t.Context(), 0, &issues_model.PullRequest{
			ID: pullRequestID,
			Issue: &issues_model.Issue{
				RepoID: repoID,
			},
		}, UserTrustDenied))

		currentCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})
		assert.Equal(t, previousCancelledCount+1, currentCancelledCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusCancelled.String(), run.Status.String())
	})

	t.Run("PullRequestApprove", func(t *testing.T) {
		pullRequestID := int64(534)
		runNotApproved := createPullRequestRun(t, pullRequestID, repoID)

		previousWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})

		doerID := int64(84322)
		require.NoError(t, PullRequestApprove(t.Context(), doerID, &issues_model.PullRequest{
			ID: pullRequestID,
			Issue: &issues_model.Issue{
				RepoID: repoID,
			},
		}))

		currentWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})
		assert.Equal(t, previousWaitingCount+1, currentWaitingCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusWaiting.String(), run.Status.String())
		assert.Equal(t, doerID, run.ApprovedBy)
		assert.False(t, run.NeedApproval)
	})

	t.Run("UpdateTrustedWithPullRequest trusted", func(t *testing.T) {
		pullRequestID := int64(534)
		runNotApproved := createPullRequestRun(t, pullRequestID, repoID)

		previousWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})

		doerID := int64(84322)
		require.NoError(t, UpdateTrustedWithPullRequest(t.Context(), doerID, &issues_model.PullRequest{
			ID: pullRequestID,
			Issue: &issues_model.Issue{
				RepoID: repoID,
			},
		}, UserTrustedOnce))

		currentWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})
		assert.Equal(t, previousWaitingCount+1, currentWaitingCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusWaiting.String(), run.Status.String())
		assert.Equal(t, doerID, run.ApprovedBy)
		assert.False(t, run.NeedApproval)
	})
}

func TestActionsTrust_LoadPullRequest(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestActionsTrust_LoadPullRequest")()
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("implicitly trusted because the pull request is not from a fork", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.False(t, pr.IsForkPullRequest())
		assert.Equal(t, PosterIsImplicitlyTrustedWithActions, trust)
	})

	t.Run("implicitly trusted because the poster of a forked pull request is admin", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 3000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		require.True(t, pr.Issue.Poster.IsAdmin)
		assert.Equal(t, PosterIsImplicitlyTrustedWithActions, trust)
	})

	t.Run("explicitly trusted because the poster of a forked pull request is trusted", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), pr.Issue.PosterID, pr.Issue.RepoID)
		require.NoError(t, err)
		assert.Equal(t, PosterIsExplicitlyTrustedWithActions, trust)
	})

	t.Run("explicitly trusted because the poster of a forked pull request was permanently approved", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), pr.Issue.PosterID, pr.Issue.RepoID)
		require.NoError(t, err)
		assert.Equal(t, PosterIsExplicitlyTrustedWithActions, trust)
	})

	t.Run("not trusted because the poster of a forked pull request has no privileges", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 4000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		assert.Equal(t, PosterIsNotTrustedWithActions, trust)
	})

	t.Run("not trusted because the poster of a forked pull request is restricted", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		trust, err := LoadPullRequestPosterIsTrustedWithActions(t.Context(), pr)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), pr.Issue.PosterID, pr.Issue.RepoID)
		require.NoError(t, err)
		require.True(t, pr.Issue.Poster.IsRestricted)
		assert.Equal(t, PosterIsNotTrustedWithActions, trust)
	})

	t.Run("approval not needed because the run is not from a fork", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: false,
		}
		needApproval, err := ifNeedApproval(t.Context(), run, nil)
		require.NoError(t, err)
		assert.False(t, needApproval)
	})

	t.Run("approval not needed because the run is a pull_request_target event", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
			TriggerEvent:      actions_module.GithubEventPullRequestTarget,
		}
		needApproval, err := ifNeedApproval(t.Context(), run, nil)
		require.NoError(t, err)
		assert.False(t, needApproval)
	})

	t.Run("approval needed because the run is from a forked pull request and the user is not trusted", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
		}
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		needApproval, err := ifNeedApproval(t.Context(), run, pr)
		require.NoError(t, err)
		require.True(t, pr.Issue.Poster.IsRestricted)
		assert.True(t, needApproval)
	})

	t.Run("run for a pull request is set with trust data", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
		}
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		require.NoError(t, SetRunTrustForPullRequest(t.Context(), run, nil))
		require.NoError(t, SetRunTrustForPullRequest(t.Context(), run, pr))
		require.True(t, pr.Issue.Poster.IsRestricted)
		assert.True(t, run.NeedApproval)
		assert.True(t, run.IsForkPullRequest)
		assert.Equal(t, pr.Issue.PosterID, run.PullRequestPosterID)
		assert.Equal(t, pr.ID, run.PullRequestID)
	})
}

func TestActionsTrust_updateTrusted(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1})
	require.NoError(t, loadPullRequestAttributes(t.Context(), pr))

	t.Run("UserTrustedOnce", func(t *testing.T) {
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
		require.NoError(t, updateTrusted(t.Context(), pr, UserTrustedOnce))
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
	})

	t.Run("UserAlwaysTrusted", func(t *testing.T) {
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
		require.NoError(t, updateTrusted(t.Context(), pr, UserAlwaysTrusted))
		require.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
	})

	t.Run("UserTrustRevoked", func(t *testing.T) {
		require.Equal(t, 1, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
		require.NoError(t, updateTrusted(t.Context(), pr, UserTrustRevoked))
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
	})

	t.Run("UserTrustDenied", func(t *testing.T) {
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
		require.NoError(t, updateTrusted(t.Context(), pr, UserTrustDenied))
		require.Zero(t, unittest.GetCount(t, &actions_model.ActionUser{UserID: pr.Issue.Poster.ID}))
	})
}
