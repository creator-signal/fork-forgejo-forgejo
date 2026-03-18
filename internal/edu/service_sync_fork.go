package edu

import (
	"context"
	"fmt"
	"strings"
	"time"
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

	doer, err := s.users.GetUserByID(ctx, doerID)
	if err != nil {
		return nil, fmt.Errorf("get doer: %w", err)
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
		Status:       "running",
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	if err := s.repo.CreateSyncForkTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create sync fork task: %w", err)
	}

	if len(submissions) == 0 {
		task.Status = "done"
		task.UpdatedUnix = time.Now().Unix()
		_ = s.repo.UpdateSyncForkTask(ctx, task)
		return task, nil
	}

	for _, sub := range submissions {
		if sub.StudentRepoID == 0 {
			task.Skipped++
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateSyncForkTask(ctx, task)
			continue
		}

		forkRepo, err := s.forker.GetRepositoryByID(ctx, sub.StudentRepoID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("repo %d: get repo: %v\n", sub.StudentRepoID, err)
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateSyncForkTask(ctx, task)
			continue
		}

		if !forkRepo.IsFork {
			task.Skipped++
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateSyncForkTask(ctx, task)
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
			_ = s.repo.UpdateSyncForkTask(ctx, task)
			continue
		}

		task.Synced++
		task.UpdatedUnix = time.Now().Unix()
		_ = s.repo.UpdateSyncForkTask(ctx, task)
	}

	if task.Failed > 0 {
		task.Status = "error"
	} else {
		task.Status = "done"
	}
	task.UpdatedUnix = time.Now().Unix()
	_ = s.repo.UpdateSyncForkTask(ctx, task)

	return task, nil
}

func (s *service) GetSyncForkTask(ctx context.Context, taskID int64) (*SyncForkTask, error) {
	return s.repo.GetSyncForkTask(ctx, taskID)
}

func (s *service) GetSyncForkTaskByAssignment(ctx context.Context, assignmentID int64) (*SyncForkTask, error) {
	return s.repo.GetSyncForkTaskByAssignment(ctx, assignmentID)
}
