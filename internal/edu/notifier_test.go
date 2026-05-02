package edu

import (
	"context"
	"testing"

	actions_model "forgejo.org/models/actions"
	"github.com/stretchr/testify/mock"
)

func TestActionRunNowDone(t *testing.T) {
	ctx := context.Background()

	t.Run("SuccessMatchesSubmission", func(t *testing.T) {
		mockRepo := new(MockRepository)
		notifier := &EduNotifier{
			repo:        mockRepo,
			gradeParser: func(_ context.Context, _ *actions_model.ActionRun) (int, bool) { return 0, false },
		}

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
		mockRepo.On("AutoGradeSubmission", ctx, int64(1), 100).Return(nil)
		mockRepo.On("CreateTestResult", ctx, mock.MatchedBy(func(tr *TestResult) bool {
			return tr.SubmissionID == submission.ID && tr.Score == 100
		})).Return(nil)

		notifier.ActionRunNowDone(ctx, run, actions_model.StatusRunning, nil)

		mockRepo.AssertExpectations(t)
	})

	t.Run("FailureMatchesSubmission", func(t *testing.T) {
		mockRepo := new(MockRepository)
		notifier := &EduNotifier{
			repo:        mockRepo,
			gradeParser: func(_ context.Context, _ *actions_model.ActionRun) (int, bool) { return 0, false },
		}

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
		mockRepo.On("AutoGradeSubmission", ctx, int64(2), 0).Return(nil)
		mockRepo.On("CreateTestResult", ctx, mock.MatchedBy(func(tr *TestResult) bool {
			return tr.SubmissionID == submission.ID && tr.Score == 0
		})).Return(nil)

		notifier.ActionRunNowDone(ctx, run, actions_model.StatusRunning, nil)

		mockRepo.AssertExpectations(t)
	})

	t.Run("NotSubmissionRepo", func(t *testing.T) {
		mockRepo := new(MockRepository)
		notifier := &EduNotifier{
			repo:        mockRepo,
			gradeParser: func(_ context.Context, _ *actions_model.ActionRun) (int, bool) { return 0, false },
		}

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
		mockRepo := new(MockRepository)
		notifier := &EduNotifier{
			repo:        mockRepo,
			gradeParser: func(_ context.Context, _ *actions_model.ActionRun) (int, bool) { return 0, false },
		}

		run := &actions_model.ActionRun{
			Status: actions_model.StatusRunning,
		}
		// Should do nothing, no repo calls
		notifier.ActionRunNowDone(ctx, run, actions_model.StatusWaiting, nil)

		mockRepo.AssertExpectations(t)
	})
}
