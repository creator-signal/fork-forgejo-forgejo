package edu

import (
	"context"
	"fmt"
	"time"
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
		Status:       "running",
		CreatedUnix:  now,
		UpdatedUnix:  now,
	}

	if err := s.repo.CreateBulkForkTask(ctx, task); err != nil {
		return nil, fmt.Errorf("create bulk fork task: %w", err)
	}

	if len(studentEnrollments) == 0 {
		task.Status = "done"
		task.UpdatedUnix = time.Now().Unix()
		_ = s.repo.UpdateBulkForkTask(ctx, task)
		return task, nil
	}

	doerUser, err := s.users.GetUserByID(ctx, doerID)
	if err != nil {
		return nil, fmt.Errorf("get doer user: %w", err)
	}

	for _, enrollment := range studentEnrollments {
		studentUser, err := s.users.GetUserByID(ctx, enrollment.UserID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("user %d: get user: %v\n", enrollment.UserID, err)
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateBulkForkTask(ctx, task)
			continue
		}

		// Check if submission already exists
		existing, err := s.repo.GetSubmission(ctx, assignmentID, enrollment.UserID)
		if err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("%s: check submission: %v\n", studentUser.Name, err)
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateBulkForkTask(ctx, task)
			continue
		}
		if existing != nil {
			task.Completed++
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateBulkForkTask(ctx, task)
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
			_ = s.repo.UpdateBulkForkTask(ctx, task)
			continue
		}

		submission := &Submission{
			AssignmentID:  assignmentID,
			UserID:        enrollment.UserID,
			StudentRepoID: forkedRepo.ID,
			Status:        "started",
			CreatedUnix:   time.Now().Unix(),
			UpdatedUnix:   time.Now().Unix(),
		}

		if err := s.repo.CreateSubmission(ctx, submission); err != nil {
			task.Failed++
			task.ErrorLog += fmt.Sprintf("%s: create submission: %v\n", studentUser.Name, err)
			task.UpdatedUnix = time.Now().Unix()
			_ = s.repo.UpdateBulkForkTask(ctx, task)
			continue
		}

		task.Completed++
		task.UpdatedUnix = time.Now().Unix()
		_ = s.repo.UpdateBulkForkTask(ctx, task)
	}

	if task.Failed > 0 {
		task.Status = "error"
	} else {
		task.Status = "done"
	}
	task.UpdatedUnix = time.Now().Unix()
	_ = s.repo.UpdateBulkForkTask(ctx, task)

	return task, nil
}

func (s *service) GetBulkForkTask(ctx context.Context, taskID int64) (*BulkForkTask, error) {
	return s.repo.GetBulkForkTask(ctx, taskID)
}

func (s *service) GetBulkForkTaskByAssignment(ctx context.Context, assignmentID int64) (*BulkForkTask, error) {
	return s.repo.GetBulkForkTaskByAssignment(ctx, assignmentID)
}
