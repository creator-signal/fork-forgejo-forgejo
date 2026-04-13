package edu

import (
	"context"
	"fmt"
	"time"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/modules/actions"
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
		return
	}

	var newStatus SubmissionStatus
	if run.Status == actions_model.StatusSuccess {
		newStatus = StatusPassed
	} else if run.Status == actions_model.StatusFailure {
		newStatus = StatusFailed
	} else {
		return
	}

	submission.Status = newStatus
	submission.UpdatedUnix = time.Now().Unix()

	if err := n.repo.UpdateSubmission(ctx, submission); err != nil {
		log.Error("EduNotifier: failed to update submission %d status: %v", submission.ID, err)
	} else {
		log.Info("EduNotifier: updated submission %d status to %s", submission.ID, newStatus)
	}

	// Try to parse grade from workflow logs
	score, foundGrade := n.parseGradeFromRun(ctx, run)
	if !foundGrade {
		// Fallback: binary scoring
		if run.Status == actions_model.StatusSuccess {
			score = 100
		} else {
			score = 0
		}
	}

	// Create TestResult record (always, for history)
	result := &TestResult{
		SubmissionID: submission.ID,
		CommitSHA:    run.CommitSHA,
		Score:        score,
		Details:      fmt.Sprintf("Workflow: %s, Status: %s", run.Title, run.Status),
		CreatedUnix:  time.Now().Unix(),
	}
	if err := n.repo.CreateTestResult(ctx, result); err != nil {
		log.Error("EduNotifier: failed to create test result for submission %d: %v", submission.ID, err)
	}

	// Update submission grade only if not manually graded
	if !submission.ManualGrade {
		if err := n.repo.AutoGradeSubmission(ctx, submission.ID, score); err != nil {
			log.Error("EduNotifier: failed to auto-grade submission %d: %v", submission.ID, err)
		} else {
			log.Info("EduNotifier: auto-graded submission %d with score %d", submission.ID, score)
		}
	}
}

// parseGradeFromRun reads all job logs for a run and extracts ::edu-grade::XX.
func (n *EduNotifier) parseGradeFromRun(ctx context.Context, run *actions_model.ActionRun) (int, bool) {
	jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
	if err != nil {
		log.Error("EduNotifier: failed to get jobs for run %d: %v", run.ID, err)
		return 0, false
	}

	var allLines []string
	for _, job := range jobs {
		if job.TaskID == 0 {
			continue
		}
		task, err := actions_model.GetTaskByID(ctx, job.TaskID)
		if err != nil {
			log.Error("EduNotifier: failed to get task %d: %v", job.TaskID, err)
			continue
		}
		if task.LogLength == 0 {
			continue
		}
		logRows, err := actions.ReadLogs(ctx, task.LogInStorage, task.LogFilename, 0, task.LogLength)
		if err != nil {
			log.Error("EduNotifier: failed to read logs for task %d: %v", task.ID, err)
			continue
		}
		for _, row := range logRows {
			allLines = append(allLines, row.Content)
		}
	}

	return ParseGradeFromLogLines(allLines)
}
