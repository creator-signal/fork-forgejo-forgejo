package edu

import (
	"context"
	"fmt"
	"time"

	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"

	repo_model "forgejo.org/models/repo"
)

func (s *service) BulkForkForAssignment(ctx context.Context, assignmentID, doerID int64) (*BulkForkTask, error) {
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

	if assignment.CourseID == 0 {
		return nil, fmt.Errorf("assignment is not bound to a course")
	}

	// Prevent duplicate concurrent tasks
	existing, _ := s.repo.GetBulkForkTaskByAssignment(ctx, assignmentID)
	if existing != nil && (existing.Status == StatusPending || existing.Status == StatusRunning) {
		return existing, nil
	}

	baseRepo, err := s.forker.GetRepositoryByID(ctx, assignment.RepoID)
	if err != nil {
		return nil, fmt.Errorf("get base repo: %w", err)
	}

	enrollments, err := s.repo.GetEnrollments(ctx, assignment.CourseID)
	if err != nil {
		return nil, fmt.Errorf("get enrollments: %w", err)
	}

	// Filter only students
	var studentEnrollments []*CourseEnrollment
	for _, e := range enrollments {
		if e.Role == RoleStudent {
			studentEnrollments = append(studentEnrollments, e)
		}
	}

	now := time.Now().Unix()
	task := &BulkForkTask{
		AssignmentID: assignmentID,
		CreatorID:    doerID,
		TotalUsers:   len(studentEnrollments),
		Status:       StatusPending,
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	if err := s.repo.CreateBulkForkTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create bulk fork task: %w", err)
	}

	if len(studentEnrollments) == 0 {
		task.Status = StatusDone
		task.UpdatedUnix = time.Now().Unix()
		if err := s.repo.UpdateBulkForkTask(ctx, task); err != nil {
			log.Error("Failed to update bulk fork task: %v", err)
		}
		return task, nil
	}

	// Start async execution
	go graceful.GetManager().RunWithShutdownContext(func(ctx context.Context) {
		s.executeBulkFork(ctx, task, assignmentID, doerID, baseRepo, studentEnrollments)
	})

	return task, nil
}

// executeBulkFork runs the actual fork loop in a background goroutine.
func (s *service) executeBulkFork(ctx context.Context, task *BulkForkTask, assignmentID, doerID int64, baseRepo *repo_model.Repository, studentEnrollments []*CourseEnrollment) {
	task.Status = StatusRunning
	task.UpdatedUnix = time.Now().Unix()
	if err := s.repo.UpdateBulkForkTask(ctx, task); err != nil {
		log.Error("Failed to update bulk fork task to running: %v", err)
		return
	}

	doerUser, err := s.users.GetUserByID(ctx, doerID)
	if err != nil {
		task.Status = StatusError
		task.ErrorLog += fmt.Sprintf("get doer: %v\n", err)
		task.UpdatedUnix = time.Now().Unix()
		if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
			log.Error("Failed to update bulk fork task: %v", errUpd)
		}
		return
	}

	for _, enrollment := range studentEnrollments {
		studentUser, err := s.users.GetUserByID(ctx, enrollment.UserID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("user %d: get user: %v\n", enrollment.UserID, err)
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update bulk fork task: %v", errUpd)
			}
			continue
		}

		// Check if submission already exists
		existing, err := s.repo.GetSubmission(ctx, assignmentID, enrollment.UserID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("%s: check submission: %v\n", studentUser.Name, err)
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update bulk fork task: %v", errUpd)
			}
			continue
		}
		if existing != nil {
			task.Completed++
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update bulk fork task: %v", errUpd)
			}
			continue
		}

		forkName := fmt.Sprintf("%s-%s", studentUser.Name, baseRepo.Name)

		forkedRepo, err := s.forker.ForkRepositoryAndUpdates(ctx, doerUser, studentUser, ForkRepoOptions{
			BaseRepo: baseRepo,
			Name:     forkName,
		})
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("%s: fork: %v\n", studentUser.Name, err)
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update bulk fork task: %v", errUpd)
			}
			continue
		}

		submission := &Submission{
			AssignmentID:  assignmentID,
			UserID:        enrollment.UserID,
			StudentRepoID: forkedRepo.ID,
			Status:        StatusSubmissionPending,
			CreatedUnix:   time.Now().Unix(),
			UpdatedUnix:   time.Now().Unix(),
		}

		if err := s.repo.CreateSubmission(ctx, submission); err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("%s: create submission: %v\n", studentUser.Name, err)
			task.UpdatedUnix = time.Now().Unix()
			if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
				log.Error("Failed to update bulk fork task: %v", errUpd)
			}
			continue
		}

		task.Completed++
		task.UpdatedUnix = time.Now().Unix()
		if errUpd := s.repo.UpdateBulkForkTask(ctx, task); errUpd != nil {
			log.Error("Failed to update bulk fork task: %v", errUpd)
		}
	}

	if task.Failed > 0 {
		task.Status = StatusError
	} else {
		task.Status = StatusDone
	}
	task.UpdatedUnix = time.Now().Unix()
	if err := s.repo.UpdateBulkForkTask(ctx, task); err != nil {
		log.Error("Failed to update bulk fork task: %v", err)
	}
}

func (s *service) GetBulkForkTask(ctx context.Context, taskID int64) (*BulkForkTask, error) {
	return s.repo.GetBulkForkTask(ctx, taskID)
}

func (s *service) GetBulkForkTaskByAssignment(ctx context.Context, assignmentID int64) (*BulkForkTask, error) {
	return s.repo.GetBulkForkTaskByAssignment(ctx, assignmentID)
}
