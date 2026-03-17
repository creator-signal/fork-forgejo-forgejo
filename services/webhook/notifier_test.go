// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webhook

import (
	"fmt"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	webhook_model "forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	"forgejo.org/modules/json"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/structs"
	"forgejo.org/modules/test"
	"forgejo.org/modules/timeutil"
	webhook_module "forgejo.org/modules/webhook"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pushCommits() *repository.PushCommits {
	pushCommits := repository.NewPushCommits()
	pushCommits.Commits = []*repository.PushCommit{
		{
			Sha1:           "2c54faec6c45d31c1abfaecdab471eac6633738a",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "not signed commit",
		},
		{
			Sha1:           "205ac761f3326a7ebe416e8673760016450b5cec",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit (with not yet validated email)",
		},
		{
			Sha1:           "1032bbf17fbc0d9c95bb5418dabe8f8c99278700",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit",
		},
	}
	pushCommits.HeadCommit = &repository.PushCommit{Sha1: "2c54faec6c45d31c1abfaecdab471eac6633738a"}
	return pushCommits
}

func TestSyncPushCommits(t *testing.T) {
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master-1")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%master-1%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 3)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 1)()

		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main-1")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%main-1%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 1)
		assert.Equal(t, "2c54faec6c45d31c1abfaecdab471eac6633738a", payloadContent.Commits[0].ID)
	})
}

func TestPushCommits(t *testing.T) {
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: user.ID})

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master-2")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%master-2%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 3)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 1)()

		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main-2")}, pushCommits())

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("payload_content LIKE '%main-2%'"))

		var payloadContent structs.PushPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Len(t, payloadContent.Commits, 1)
		assert.Equal(t, "2c54faec6c45d31c1abfaecdab471eac6633738a", payloadContent.Commits[0].ID)
	})
}

func assertActionEqual(t *testing.T, expectedRun *actions_model.ActionRun, actualRun *structs.ActionRun) {
	assert.NotNil(t, expectedRun)
	assert.NotNil(t, actualRun)
	// only test a few things
	assert.Equal(t, expectedRun.ID, actualRun.ID)
	assert.Equal(t, expectedRun.Status.String(), actualRun.Status)
	assert.Equal(t, expectedRun.Index, actualRun.Index)
	assert.Equal(t, expectedRun.RepoID, actualRun.Repo.ID)
	// convert to unix because of time zones
	assert.Equal(t, expectedRun.Stopped.AsTime().Unix(), actualRun.Stopped.Unix())
	assert.Equal(t, expectedRun.Title, actualRun.Title)
	assert.Equal(t, expectedRun.WorkflowID, actualRun.WorkflowID)
}

func TestAction(t *testing.T) {
	defer unittest.OverrideFixtures("services/webhook/TestPushCommits")()
	require.NoError(t, unittest.PrepareTestDatabase())

	triggerUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2, OwnerID: triggerUser.ID, Owner: triggerUser})

	oldSuccessRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusSuccess,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648027,
		WorkflowID:    "some_workflow",
		Title:         "oldSuccessRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	oldSuccessRun.LoadAttributes(db.DefaultContext)
	oldFailureRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusFailure,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648027,
		WorkflowID:    "some_workflow",
		Title:         "oldFailureRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	oldFailureRun.LoadAttributes(db.DefaultContext)
	newSuccessRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusSuccess,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648327,
		WorkflowID:    "some_workflow",
		Title:         "newSuccessRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	newSuccessRun.LoadAttributes(db.DefaultContext)
	newFailureRun := &actions_model.ActionRun{
		ID:            1,
		Status:        actions_model.StatusFailure,
		Index:         1,
		RepoID:        repo.ID,
		Stopped:       1693648327,
		WorkflowID:    "some_workflow",
		Title:         "newFailureRun",
		TriggerUser:   triggerUser,
		TriggerUserID: triggerUser.ID,
		TriggerEvent:  "push",
	}
	newFailureRun.LoadAttributes(db.DefaultContext)

	t.Run("Successful Run after Nothing", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, nil)

		// there's only one of these at the time
		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_success' AND payload_content LIKE '%success%newSuccessRun%'"))
		assert.Equal(t, webhook_module.HookEventActionRunSuccess, hookTask.EventType)

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, structs.HookActionSuccess, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newSuccessRun, payloadContent.Run)
		assert.Nil(t, payloadContent.LastRun)
	})

	t.Run("Successful Run after Failure", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, oldFailureRun)

		{
			hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_success' AND payload_content LIKE '%success%newSuccessRun%oldFailureRun%'"))
			assert.Equal(t, webhook_module.HookEventActionRunSuccess, hookTask.EventType)

			var payloadContent structs.ActionPayload
			require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
			assert.Equal(t, structs.HookActionSuccess, payloadContent.Action)
			assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
			assertActionEqual(t, newSuccessRun, payloadContent.Run)
			assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
		}
		{
			hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_recover' AND payload_content LIKE '%recover%newSuccessRun%oldFailureRun%'"))
			assert.Equal(t, webhook_module.HookEventActionRunRecover, hookTask.EventType)

			log.Error("something: %s", hookTask.PayloadContent)
			var payloadContent structs.ActionPayload
			require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
			assert.Equal(t, structs.HookActionRecover, payloadContent.Action)
			assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
			assertActionEqual(t, newSuccessRun, payloadContent.Run)
			assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
		}
	})

	t.Run("Successful Run after Success", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newSuccessRun, actions_model.StatusWaiting, oldSuccessRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_success' AND payload_content LIKE '%success%newSuccessRun%oldSuccessRun%'"))
		assert.Equal(t, webhook_module.HookEventActionRunSuccess, hookTask.EventType)

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, structs.HookActionSuccess, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newSuccessRun, payloadContent.Run)
		assertActionEqual(t, oldSuccessRun, payloadContent.LastRun)
	})

	t.Run("Failed Run after Nothing", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, nil)

		// there should only be this one at the time
		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_failure' AND payload_content LIKE '%failure%newFailureRun%'"))
		assert.Equal(t, webhook_module.HookEventActionRunFailure, hookTask.EventType)

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, structs.HookActionFailure, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assert.Nil(t, payloadContent.LastRun)
	})

	t.Run("Failed Run after Failure", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, oldFailureRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_failure' AND payload_content LIKE '%failure%newFailureRun%oldFailureRun%'"))
		assert.Equal(t, webhook_module.HookEventActionRunFailure, hookTask.EventType)

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, structs.HookActionFailure, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assertActionEqual(t, oldFailureRun, payloadContent.LastRun)
	})

	t.Run("Failed Run after Success", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		NewNotifier().ActionRunNowDone(db.DefaultContext, newFailureRun, actions_model.StatusWaiting, oldSuccessRun)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'action_run_failure' AND payload_content LIKE '%failure%newFailureRun%oldSuccessRun%'"))
		assert.Equal(t, webhook_module.HookEventActionRunFailure, hookTask.EventType)

		var payloadContent structs.ActionPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))
		assert.Equal(t, structs.HookActionFailure, payloadContent.Action)
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.PriorStatus)
		assertActionEqual(t, newFailureRun, payloadContent.Run)
		assertActionEqual(t, oldSuccessRun, payloadContent.LastRun)
	})

	t.Run("Workflow Job Status update - nil task", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		now := timeutil.TimeStampNow()
		nilTaskJob := &actions_model.ActionRunJob{
			ID:      999,
			RunID:   oldSuccessRun.ID,
			Run:     oldSuccessRun,
			RepoID:  repo.ID,
			OwnerID: triggerUser.ID,
			Status:  actions_model.StatusWaiting,
			Name:    "nil-task-job",
			RunsOn:  []string{"ubuntu-latest"},
			Attempt: 1,
			Created: now,
			Started: now,
			Stopped: now,
		}

		NewNotifier().WorkflowJobStatusUpdate(db.DefaultContext, repo, triggerUser, nilTaskJob, nil)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'workflow_job' AND payload_content LIKE '%nil-task-job%'"))
		assert.Equal(t, webhook_module.HookEventWorkflowJob, hookTask.EventType)

		var payloadContent structs.WorkflowJobPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))

		// Steps must be an empty array (not null) when task is nil
		assert.NotNil(t, payloadContent.WorkflowJob.Steps)
		assert.Empty(t, payloadContent.WorkflowJob.Steps)
		assert.Contains(t, hookTask.PayloadContent, `"steps": []`)
	})

	t.Run("Workflow Job Status update", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Webhook.PayloadCommitLimit, 10)()

		now := timeutil.TimeStampNow()
		actionRunJob := &actions_model.ActionRunJob{
			ID:      1,
			RunID:   oldSuccessRun.ID,
			Run:     oldSuccessRun,
			RepoID:  repo.ID,
			OwnerID: triggerUser.ID,
			Status:  actions_model.StatusWaiting,
			Name:    "test-job",
			RunsOn:  []string{"ubuntu-latest"},
			Attempt: 1,
			Created: now,
			Started: now,
			Stopped: now,
		}

		actionRunner := &actions_model.ActionRunner{
			ID:   10,
			Name: "test-runner",
		}
		require.NoError(t, db.Insert(db.DefaultContext, actionRunner))

		actionTask := &actions_model.ActionTask{
			ID:       1,
			JobID:    actionRunJob.ID,
			Status:   actions_model.StatusFailure,
			RunnerID: actionRunner.ID,
		}
		// Create task steps
		taskSteps := []*actions_model.ActionTaskStep{
			{
				Name:    "checkout",
				TaskID:  actionTask.ID,
				Index:   0,
				Status:  actions_model.StatusSuccess,
				RepoID:  repo.ID,
				Started: now,
				Stopped: now,
			},
			{
				Name:    "build",
				TaskID:  actionTask.ID,
				Index:   1,
				Status:  actions_model.StatusFailure,
				RepoID:  repo.ID,
				Started: now,
				Stopped: now,
			},
		}
		actionTask.Steps = taskSteps

		NewNotifier().WorkflowJobStatusUpdate(db.DefaultContext, repo, triggerUser, actionRunJob, actionTask)

		hookTask := unittest.AssertExistsAndLoadBean(t, &webhook_model.HookTask{}, unittest.Cond("event_type == 'workflow_job' AND payload_content LIKE '%test-job%'"))
		assert.Equal(t, webhook_module.HookEventWorkflowJob, hookTask.EventType)

		var payloadContent structs.WorkflowJobPayload
		require.NoError(t, json.Unmarshal([]byte(hookTask.PayloadContent), &payloadContent))

		// Test Action and basic fields
		assert.Equal(t, actions_model.StatusWaiting.String(), payloadContent.Action)
		assert.NotNil(t, payloadContent.WorkflowJob)
		assert.Equal(t, actionRunJob.ID, payloadContent.WorkflowJob.ID)
		assert.Equal(t, actionRunJob.RunID, payloadContent.WorkflowJob.RunID)
		assert.Equal(t, actionRunJob.Name, payloadContent.WorkflowJob.Name)
		assert.Equal(t, actionRunJob.RunsOn, payloadContent.WorkflowJob.Labels)
		assert.Equal(t, actionRunJob.Attempt, payloadContent.WorkflowJob.RunAttempt)

		// Test URLs
		assert.Contains(t, payloadContent.WorkflowJob.RunURL, fmt.Sprintf("/actions/runs/%d", actionRunJob.RunID))

		// Test commit info
		assert.Equal(t, oldSuccessRun.CommitSHA, payloadContent.WorkflowJob.HeadSha)
		assert.Equal(t, git.RefName(oldSuccessRun.Ref).BranchName(), payloadContent.WorkflowJob.HeadBranch)

		// Test runner info
		assert.Equal(t, actionRunner.ID, payloadContent.WorkflowJob.RunnerID)
		assert.Equal(t, actionRunner.Name, payloadContent.WorkflowJob.RunnerName)

		// Test steps
		require.Len(t, payloadContent.WorkflowJob.Steps, 2)
		assert.Equal(t, "checkout", payloadContent.WorkflowJob.Steps[0].Name)
		assert.Equal(t, int64(0), payloadContent.WorkflowJob.Steps[0].Number)
		assert.Equal(t, actions_model.StatusSuccess.String(), payloadContent.WorkflowJob.Steps[0].Status)

		assert.Equal(t, "build", payloadContent.WorkflowJob.Steps[1].Name)
		assert.Equal(t, int64(1), payloadContent.WorkflowJob.Steps[1].Number)
		assert.Equal(t, actions_model.StatusFailure.String(), payloadContent.WorkflowJob.Steps[1].Status)

		// Test timestamps
		assert.Equal(t, optional.FromNonDefault(actionRunJob.Created.AsTime().UTC()), payloadContent.WorkflowJob.CreatedAt)
		assert.Equal(t, optional.FromNonDefault(actionRunJob.Started.AsTime().UTC()), payloadContent.WorkflowJob.StartedAt)
		assert.Equal(t, optional.FromNonDefault(actionRunJob.Stopped.AsTime().UTC()), payloadContent.WorkflowJob.CompletedAt)

		// Test repository info
		assert.NotNil(t, payloadContent.Repo)
		assert.Equal(t, repo.ID, payloadContent.Repo.ID)
		assert.Equal(t, repo.Name, payloadContent.Repo.Name)
		assert.Equal(t, repo.FullName(), payloadContent.Repo.FullName)
	})
}
