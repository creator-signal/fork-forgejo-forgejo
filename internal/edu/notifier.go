package edu

import (
	"context"
	"fmt"
	"time"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/log"
	"forgejo.org/services/notify"
)

type EduNotifier struct {
	notify.NullNotifier
	repo Repository
}

var _ notify.Notifier = &EduNotifier{}

func RegisterNotifier(repo Repository) {
	notify.RegisterNotifier(&EduNotifier{repo: repo})
}

func (n *EduNotifier) ActionRunNowDone(
	ctx context.Context,
	run *actions_model.ActionRun,
	priorStatus actions_model.Status,
	lastRun *actions_model.ActionRun,
) {
	if !run.Status.IsDone() {
		return
	}

	submission, err := n.repo.GetSubmissionByRepoID(ctx, run.RepoID)
	if err != nil {
		log.Error("EduNotifier: failed to get submission for repo %d: %v", run.RepoID, err)
		return
	}
	if submission == nil {
		// Not a student submission repo
		return
	}

	var newStatus SubmissionStatus
	if run.Status == actions_model.StatusSuccess {
		newStatus = StatusPassed
	} else if run.Status == actions_model.StatusFailure {
		newStatus = StatusFailed
	} else {
		return // TODO: handle cancelled, skipped, etc.
	}

	submission.Status = newStatus
	submission.UpdatedUnix = time.Now().Unix()

	if err := n.repo.UpdateSubmission(ctx, submission); err != nil {
		log.Error("EduNotifier: failed to update submission %d status: %v", submission.ID, err)
	} else {
		log.Info("EduNotifier: updated submission %d status to %s", submission.ID, newStatus)
	}

	// Create TestResult record
	result := &TestResult{
		SubmissionID: submission.ID,
		CommitSHA:    run.CommitSHA,
		Score:        0,
		Details:      fmt.Sprintf("Workflow: %s, Status: %s", run.Title, run.Status),
		CreatedUnix:  time.Now().Unix(),
	}
	if run.Status == actions_model.StatusSuccess {
		result.Score = 100
	}

	if err := n.repo.CreateTestResult(ctx, result); err != nil {
		log.Error("EduNotifier: failed to create test result for submission %d: %v", submission.ID, err)
	}
}
