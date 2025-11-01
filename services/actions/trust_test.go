// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
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

	t.Run("RevokeTrust", func(t *testing.T) {
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

		require.NoError(t, RevokeTrust(t.Context(), repoID, pullRequestPosterID))

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

		require.NoError(t, PullRequestCancel(t.Context(), repoID, pullRequestID))

		currentCancelledCount := unittest.GetCount(t, &actions_model.ActionRun{Status: actions_model.StatusCancelled})
		assert.Equal(t, previousCancelledCount+1, currentCancelledCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusCancelled.String(), run.Status.String())
	})

	t.Run("UpdateTrustedWithPullRequest deny", func(t *testing.T) {
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
		require.NoError(t, PullRequestApprove(t.Context(), doerID, repoID, pullRequestID))

		currentWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})
		assert.Equal(t, previousWaitingCount+1, currentWaitingCount)
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: runNotApproved.ID})
		assert.Equal(t, actions_model.StatusWaiting.String(), run.Status.String())
		assert.Equal(t, doerID, run.ApprovedBy)
		assert.False(t, run.NeedApproval)
	})

	t.Run("UpdateTrustedWithPullRequest once", func(t *testing.T) {
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

	t.Run("UpdateTrustedWithPullRequest always", func(t *testing.T) {
		pullRequestIDs := []int64{534, 645}
		var runsNotApproved []*actions_model.ActionRun
		for _, pullRequestID := range pullRequestIDs {
			runsNotApproved = append(runsNotApproved, createPullRequestRun(t, pullRequestID, repoID))
		}

		previousWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})

		doerID := int64(84322)
		require.NoError(t, UpdateTrustedWithPullRequest(t.Context(), doerID, &issues_model.PullRequest{
			ID: pullRequestIDs[0],
			Issue: &issues_model.Issue{
				RepoID:   repoID,
				PosterID: pullRequestPosterID,
			},
		}, UserAlwaysTrusted))

		currentWaitingCount := unittest.GetCount(t, &actions_model.ActionRunJob{Status: actions_model.StatusWaiting})
		assert.Equal(t, previousWaitingCount+len(pullRequestIDs), currentWaitingCount)

		for _, run := range runsNotApproved {
			run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: run.ID})
			assert.Equal(t, actions_model.StatusWaiting.String(), run.Status.String())
			assert.Equal(t, doerID, run.ApprovedBy)
			assert.False(t, run.NeedApproval)
		}
	})

	t.Run("UpdateTrustedWithPullRequest revoke", func(t *testing.T) {
		pullRequestIDs := []int64{748, 953}
		var runsNotApproved []*actions_model.ActionRun
		for _, pullRequestID := range pullRequestIDs {
			runsNotApproved = append(runsNotApproved, createPullRequestRun(t, pullRequestID, repoID))
		}

		doerID := int64(84322)
		require.NoError(t, UpdateTrustedWithPullRequest(t.Context(), doerID, &issues_model.PullRequest{
			ID: pullRequestIDs[0],
			Issue: &issues_model.Issue{
				RepoID:   repoID,
				PosterID: pullRequestPosterID,
			},
		}, UserTrustRevoked))

		for _, run := range runsNotApproved {
			run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: run.ID})
			assert.Equal(t, actions_model.StatusCancelled.String(), run.Status.String())
			assert.False(t, run.NeedApproval)
		}
	})
}

func TestActionsTrust_GetPullRequestUserIsTrustedWithActions(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestActionsTrust_GetPullRequestUserIsTrustedWithActions")()
	require.NoError(t, unittest.PrepareTestDatabase())

	t.Run("implicitly trusted because the pull request is not from a fork", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2000})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // regular user
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, user)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.False(t, pr.IsForkPullRequest())
		assert.Equal(t, UserIsImplicitlyTrustedWithActions, trust)
	})

	t.Run("implicitly trusted on a forked pull request when the user admin", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 3000})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}) // admin
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, user)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		require.True(t, user.IsAdmin)
		assert.Equal(t, UserIsImplicitlyTrustedWithActions, trust)
	})

	t.Run("explicitly trusted on a forked pull request when the user was permanently approved", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1000})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4}) // regular user
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, user)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), user.ID, pr.Issue.RepoID)
		require.NoError(t, err)
		assert.Equal(t, UserIsExplicitlyTrustedWithActions, trust)
	})

	t.Run("not trusted because on a forked pull request when the user has has no privileges", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 4000})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 5}) // regular user
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, user)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		assert.Equal(t, UserIsNotTrustedWithActions, trust)
	})

	t.Run("not trusted on a forked pull request because the user is restricted", func(t *testing.T) {
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29}) // restricted user
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, user)
		require.NoError(t, err)
		pr.LoadAttributes(t.Context())
		require.True(t, pr.IsForkPullRequest())
		_, err = actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(t.Context(), user.ID, pr.Issue.RepoID)
		require.NoError(t, err)
		require.True(t, user.IsRestricted)
		assert.Equal(t, UserIsNotTrustedWithActions, trust)
	})

	t.Run("approval not needed because the run is not from a fork", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: false,
		}
		needApproval, err := ifNeedApproval(t.Context(), run, nil, nil)
		require.NoError(t, err)
		assert.False(t, needApproval)
	})

	t.Run("approval not needed because the run is a pull_request_target event", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
			TriggerEvent:      actions_module.GithubEventPullRequestTarget,
		}
		needApproval, err := ifNeedApproval(t.Context(), run, nil, nil)
		require.NoError(t, err)
		assert.False(t, needApproval)
	})

	t.Run("approval needed because the run is from a forked pull request and the user is not trusted", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
		}
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29}) // restricted user
		needApproval, err := ifNeedApproval(t.Context(), run, pr, doer)
		require.NoError(t, err)
		require.True(t, doer.IsRestricted)
		assert.True(t, needApproval)
	})

	t.Run("approval needed because the run is triggered by an untrusted user from a forked pull request authored by a trusted user", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
		}
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1000})
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29}) // restricted user
		needApproval, err := ifNeedApproval(t.Context(), run, pr, doer)
		require.NoError(t, err)
		require.True(t, doer.IsRestricted)
		assert.False(t, needApproval)
		require.NoError(t, pr.LoadAttributes(t.Context()))
		poster, err := mustGetIssuePoster(t.Context(), pr.Issue)
		require.NoError(t, err)
		trust, err := GetPullRequestUserIsTrustedWithActions(t.Context(), pr, poster)
		require.NoError(t, err)
		assert.Equal(t, UserIsExplicitlyTrustedWithActions, trust)
	})

	t.Run("run for a pull request is set with info related to trust", func(t *testing.T) {
		run := &actions_model.ActionRun{
			IsForkPullRequest: true,
		}
		pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 5000})
		doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 29}) // restricted user
		require.NoError(t, SetRunTrustForPullRequest(t.Context(), run, nil, doer))
		require.NoError(t, SetRunTrustForPullRequest(t.Context(), run, pr, doer))
		require.True(t, doer.IsRestricted)
		assert.True(t, run.NeedApproval)
		assert.True(t, run.IsForkPullRequest)
		assert.Equal(t, pr.Issue.PosterID, run.PullRequestPosterID)
		assert.Equal(t, pr.ID, run.PullRequestID)
	})
}
