package edu

import (
	"context"
	"testing"

	actions_model "forgejo.org/models/actions"
	"github.com/stretchr/testify/mock"
)

func TestActionRunNowDone(t *testing.T) {
	mockRepo := new(MockRepository)
	notifier := &EduNotifier{repo: mockRepo}
	ctx := context.Background()

	t.Run("SuccessMatchesSubmission", func(t *testing.T) {
		repoID := int64(100)
		run := &actions_model.ActionRun{
			RepoID: repoID,
			Status: actions_model.StatusSuccess,
		}

		submission := &Submission{
			ID:            1,
			StudentRepoID: repoID,
			Status:        "started",
		}

		mockRepo.On("GetSubmissionByRepoID", ctx, repoID).Return(submission, nil)
		mockRepo.On("UpdateSubmission", ctx, mock.MatchedBy(func(s *Submission) bool {
			return s.ID == submission.ID && s.Status == StatusPassed
		})).Return(nil)

		notifier.ActionRunNowDone(ctx, run, actions_model.StatusRunning, nil)

		mockRepo.AssertExpectations(t)
	})

	t.Run("FailureMatchesSubmission", func(t *testing.T) {
		repoID := int64(101)
		run := &actions_model.ActionRun{
			RepoID: repoID,
			Status: actions_model.StatusFailure,
		}

		submission := &Submission{
			ID:            2,
			StudentRepoID: repoID,
			Status:        StatusStarted,
		}

		mockRepo.On("GetSubmissionByRepoID", ctx, repoID).Return(submission, nil)
		mockRepo.On("UpdateSubmission", ctx, mock.MatchedBy(func(s *Submission) bool {
			return s.ID == submission.ID && s.Status == StatusFailed
		})).Return(nil)

		notifier.ActionRunNowDone(ctx, run, actions_model.StatusRunning, nil)

		mockRepo.AssertExpectations(t)
	})

	t.Run("NotSubmissionRepo", func(t *testing.T) {
		repoID := int64(999)
		run := &actions_model.ActionRun{
			RepoID: repoID,
			Status: actions_model.StatusSuccess,
		}

		mockRepo.On("GetSubmissionByRepoID", ctx, repoID).Return(nil, nil)

		notifier.ActionRunNowDone(ctx, run, actions_model.StatusRunning, nil)

		mockRepo.AssertExpectations(t)
	})

	t.Run("IgnoreNotDone", func(t *testing.T) {
		run := &actions_model.ActionRun{
			Status: actions_model.StatusRunning,
		}
		// Should do nothing, no repo calls
		notifier.ActionRunNowDone(ctx, run, actions_model.StatusWaiting, nil)
	})
}
