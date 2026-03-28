package edu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"forgejo.org/models/db"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
)

func (s *service) SyncAllForks(ctx context.Context, assignmentID, doerID int64) (*SyncForkTask, error) {
	if s.users == nil {
		return nil, fmt.Errorf("user creator not configured")
	}

	assignment, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if assignment == nil {
		return nil, fmt.Errorf("assignment not found")
	}

	branch, err := s.forker.GetDefaultBranch(ctx, assignment.RepoID)
	if err != nil {
		return nil, fmt.Errorf("get default branch: %w", err)
	}

	submissions, err := s.repo.GetSubmissions(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("get submissions: %w", err)
	}

	now := time.Now().Unix()
	task := &SyncForkTask{
		AssignmentID: assignmentID,
		CreatorID:    doerID,
		TotalRepos:   len(submissions),
		Status:       StatusPending,
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	if err := s.repo.CreateSyncForkTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create sync fork task: %w", err)
	}

	if len(submissions) == 0 {
		task.Status = StatusDone
		task.UpdatedUnix = time.Now().Unix()
		if err := s.repo.UpdateSyncForkTask(ctx, task); err != nil {
			log.Error("Failed to update sync fork task: %v", err)
		}
		return task, nil
	}

	// Start async execution
	go graceful.GetManager().RunWithShutdownContext(func(_ context.Context) {
		s.executeSyncAllForks(db.DefaultContext, task, doerID, submissions, branch)
	})

	return task, nil
}

// executeSyncAllForks runs the actual sync loop in a background goroutine.
func (s *service) executeSyncAllForks(ctx context.Context, task *SyncForkTask, doerID int64, submissions []*Submission, branch string) {
	task.Status = StatusRunning
	task.UpdatedUnix = time.Now().Unix()
	if err := s.repo.UpdateSyncForkTask(ctx, task); err != nil {
		log.Error("Failed to update sync fork task to running: %v", err)
		return
	}

	doer, err := s.users.GetUserByID(ctx, doerID)
	if err != nil {
		task.Status = StatusError
		task.ErrorLog += fmt.Sprintf("get doer: %v\n", err)
		task.UpdatedUnix = time.Now().Unix()
		if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
			log.Error("Failed to update sync fork task: %v", errUpd)
		}
		return
	}

	for _, sub := range submissions {
		if sub.StudentRepoID == 0 {
			task.Skipped++
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update sync fork task: %v", errUpd)
			}
			continue
		}

		forkRepo, err := s.forker.GetRepositoryByID(ctx, sub.StudentRepoID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("repo %d: get repo: %v\n", sub.StudentRepoID, err)
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update sync fork task: %v", errUpd)
			}
			continue
		}

		if !forkRepo.IsFork {
			task.Skipped++
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update sync fork task: %v", errUpd)
			}
			continue
		}

		err = s.forker.SyncFork(ctx, doer, forkRepo, branch)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "already up-to-date") || strings.Contains(errStr, "up to date") {
				task.Skipped++
			} else {
				task.Failed++
				task.ErrorLog += fmt.Sprintf("%s: sync: %v\n", forkRepo.Name, err)
			}
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update sync fork task: %v", errUpd)
			}
			continue
		}

		task.Synced++
		task.UpdatedUnix = time.Now().Unix()
		if errUpd := s.repo.UpdateSyncForkTask(ctx, task); errUpd != nil {
			log.Error("Failed to update sync fork task: %v", errUpd)
		}
	}

	if task.Failed > 0 {
		task.Status = StatusError
	} else {
		task.Status = StatusDone
	}
	task.UpdatedUnix = time.Now().Unix()
	if err := s.repo.UpdateSyncForkTask(ctx, task); err != nil {
		log.Error("Failed to update sync fork task: %v", err)
	}
}

func (s *service) GetSyncForkTask(ctx context.Context, taskID int64) (*SyncForkTask, error) {
	return s.repo.GetSyncForkTask(ctx, taskID)
}

func (s *service) GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error) {
	return s.repo.GetSyncForkTaskByAssignment(ctx, assignmentID)
}
