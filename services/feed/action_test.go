// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package feed

import (
	"context"
	"strings"
	"testing"
	"time"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/queue"
	"forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func testCreateNotificationQueue(t *testing.T) func() {
	ctx, cancel := context.WithCancel(context.Background())
	q := createNotificationQueue(ctx)
	require.NotNil(t, q)
	go q.Run()
	return func() {
		// if the queue is not shutdown, goroutines will leak after the test
		// is complete and modify the database, interfering with their
		// expectations
		cancel()
		q.ShutdownWait(30 * time.Second)
		// the levelqueue is not reset in between tests, it needs to be emptied
		// before returning so it does not leak to the other tests
		for _, mq := range queue.GetManager().ManagedQueues() {
			mq.RemoveAllItems(t.Context())
		}
	}
}

func TestRenameRepoAction(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})
	defer testCreateNotificationQueue(t)()
	repo.Owner = user

	oldRepoName := repo.Name
	const newRepoName = "newRepoName"
	repo.Name = newRepoName
	repo.LowerName = strings.ToLower(newRepoName)

	actionBean := &activities_model.Action{
		OpType:    activities_model.ActionRenameRepo,
		ActUserID: user.ID,
		ActUser:   user,
		RepoID:    repo.ID,
		Repo:      repo,
		IsPrivate: repo.IsPrivate,
		Content:   oldRepoName,
	}
	unittest.AssertNotExistsBean(t, actionBean)

	NewNotifier().RenameRepository(db.DefaultContext, user, repo, oldRepoName)

	unittest.AssertExistsAndLoadBeanEventually(t, 30*time.Second, actionBean)
	unittest.CheckConsistencyFor(t, &activities_model.Action{})
}

func pushCommits() *repository.PushCommits {
	pushCommits := repository.NewPushCommits()
	pushCommits.Commits = []*repository.PushCommit{
		{
			Sha1:           "69554a6",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "not signed commit\nline two",
		},
		{
			Sha1:           "27566bd",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit (with not yet validated email)",
		},
		{
			Sha1:           "5099b81",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User2",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User2",
			Message:        "good signed commit\nlong commit message\nwith lots of details\nabout how cool the implementation is",
		},
	}
	pushCommits.HeadCommit = &repository.PushCommit{Sha1: "69554a6", Message: "not signed commit\nline two"}
	return pushCommits
}

func TestSyncPushCommits(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})
	defer testCreateNotificationQueue(t)()

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 10)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBeanEventually(t, 30*time.Second, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/master"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"27566bd","Message":"good signed commit (with not yet validated email)","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"5099b81","Message":"good signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBeanEventually(t, 30*time.Second, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/main"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Does not mutate commits param", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		commits := pushCommits()

		assert.Equal(t, "not signed commit\nline two", commits.HeadCommit.Message)
		assert.Equal(t, "good signed commit\nlong commit message\nwith lots of details\nabout how cool the implementation is", commits.Commits[2].Message)

		NewNotifier().SyncPushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master")}, commits)

		// commits passed into SyncPushCommits may be passed into other notifiers, so checking that the struct wasn't
		// mutated by truncate of messages, or truncation to match FeedMaxCommitNum (Commits[2])...
		assert.Equal(t, "not signed commit\nline two", commits.HeadCommit.Message)
		assert.Equal(t, "good signed commit\nlong commit message\nwith lots of details\nabout how cool the implementation is", commits.Commits[2].Message)
	})
}

func TestPushCommits(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})
	defer testCreateNotificationQueue(t)()

	t.Run("All commits", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 10)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("master")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBeanEventually(t, 30*time.Second, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/master"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"27566bd","Message":"good signed commit (with not yet validated email)","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},{"Sha1":"5099b81","Message":"good signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Only one commit", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		maxID := unittest.GetCount(t, &activities_model.Action{})
		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main")}, pushCommits())

		newNotification := unittest.AssertExistsAndLoadBeanEventually(t, 30*time.Second, &activities_model.Action{ActUserID: user.ID, RefName: "refs/heads/main"}, unittest.Cond("id > ?", maxID))
		assert.JSONEq(t, `{"Commits":[{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"user2@example.com","AuthorName":"User2","CommitterEmail":"user2@example.com","CommitterName":"User2","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"}],"HeadCommit":{"Sha1":"69554a6","Message":"not signed commit","AuthorEmail":"","AuthorName":"","CommitterEmail":"","CommitterName":"","Signature":null,"Verification":null,"Timestamp":"0001-01-01T00:00:00Z"},"CompareURL":"","Len":0}`, newNotification.Content)
	})

	t.Run("Does not mutate commits param", func(t *testing.T) {
		defer test.MockVariableValue(&setting.UI.FeedMaxCommitNum, 1)()

		commits := pushCommits()

		assert.Equal(t, "not signed commit\nline two", commits.HeadCommit.Message)
		assert.Equal(t, "good signed commit\nlong commit message\nwith lots of details\nabout how cool the implementation is", commits.Commits[2].Message)

		NewNotifier().PushCommits(db.DefaultContext, user, repo, &repository.PushUpdateOptions{RefFullName: git.RefNameFromBranch("main")}, commits)

		// commits passed into SyncPushCommits may be passed into other notifiers, so checking that the struct wasn't
		// mutated by truncate of messages, or truncation to match FeedMaxCommitNum (Commits[2])...
		assert.Equal(t, "not signed commit\nline two", commits.HeadCommit.Message)
		assert.Equal(t, "good signed commit\nlong commit message\nwith lots of details\nabout how cool the implementation is", commits.Commits[2].Message)
	})
}

func TestAbbreviatedComment(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "short single line comment",
			input:    "This is a short comment",
			expected: "This is a short comment",
		},
		{
			name:     "empty comment",
			input:    "",
			expected: "",
		},
		{
			name:     "multiline comment - only first line",
			input:    "First line of comment\nSecond line\nThird line",
			expected: "First line of comment",
		},
		{
			name:     "before clip boundary",
			input:    strings.Repeat("abc ", 50),
			expected: strings.Repeat("abc ", 50),
		},
		{
			name:     "after clip boundary",
			input:    strings.Repeat("abc ", 51),
			expected: "abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc abc…",
		},
		{
			name:     "byte-split would land in middle of a rune",
			input:    strings.Repeat("🎉", 200),
			expected: "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉…",
		},
		{
			name:     "mermaid block",
			input:    "Interesting point, here's a digram with my thoughts:\n```mermaid\ngraph LR\n   a -->|some text| b\n```",
			expected: "Interesting point, here's a digram with my thoughts:",
		},
		{
			name:     "block start",
			input:    "```\n# This file describes the expected reviewers for a PR based on the changed\n# files.\n```\n\nI think this comment is wrong...",
			expected: "",
		},
		{
			name:     "labeled block start",
			input:    "```mermaid\ngraph LR\n   a -->|some text| b\n```",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := abbreviatedComment(tt.input)
			assert.Equal(t, tt.expected, result, "abbreviatedComment(%q)", tt.input)
		})
	}
}

func TestDeliverNotification(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	defer test.MockVariableValue(&setting.Federation.Enabled, true)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{OwnerID: user.ID})
	require.False(t, repo.IsPrivate)

	unittest.AssertNotExistsBean(t, &repo_model.Repository{ID: 99999})
	unittest.AssertNotExistsBean(t, &user_model.User{ID: 99999})

	action := QueueableAction{
		OpType: activities_model.ActionStarRepo,
		RepoID: 99999,
	}

	notificationQueueItem := notificationQueueItem{
		Action:          action,
		LocalCount:      0,
		LocalOut:        optional.None[[]QueueableAction](),
		FederationCount: 0,
	}

	require.Error(t, deliverNotification(&notificationQueueItem))
	assert.Equal(t, uint(1), notificationQueueItem.LocalCount)
	assert.Equal(t, uint(0), notificationQueueItem.FederationCount)
	assert.False(t, notificationQueueItem.LocalOut.Has())

	// Manually set local out to pretend that the notification has been sent to
	// local watchers. Since NotifyActivityPubFollowers continues on missing
	// repositories, set the repo and use an invalid user instead.
	action.RepoID = repo.ID
	action.ActUserID = 99999

	notificationQueueItem.LocalOut = optional.Some[[]QueueableAction]([]QueueableAction{action})
	require.Error(t, deliverNotification(&notificationQueueItem))
	assert.Equal(t, uint(1), notificationQueueItem.LocalCount)
	assert.Equal(t, uint(1), notificationQueueItem.FederationCount)

	// Reset the local out and deliver the notification.
	notificationQueueItem.Action = QueueableAction{
		OpType:    activities_model.ActionStarRepo,
		RepoID:    repo.ID,
		ActUserID: user.ID,
	}

	notificationQueueItem.LocalOut = optional.None[[]QueueableAction]()

	require.NoError(t, deliverNotification(&notificationQueueItem))
	assert.Equal(t, uint(1), notificationQueueItem.LocalCount)
	assert.Equal(t, uint(1), notificationQueueItem.FederationCount)
	assert.True(t, notificationQueueItem.LocalOut.Has())
}
