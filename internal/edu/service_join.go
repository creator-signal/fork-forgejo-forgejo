package edu

import (
	"context"
	"fmt"
	"time"

	user_model "forgejo.org/models/user"
)

func (s *service) JoinAssignment(ctx context.Context, doer *user_model.User, assignmentID int64) (*Submission, error) {
	assignment, err := s.repo.GetAssignmentByID(ctx, assignmentID)
	if err != nil {
		return nil, fmt.Errorf("get assignment: %w", err)
	}
	if assignment == nil {
		return nil, fmt.Errorf("assignment not found")
	}

	if assignment.DeadlineUnix > 0 && time.Now().Unix() > assignment.DeadlineUnix {
		return nil, fmt.Errorf("assignment deadline has passed")
	}

	if assignment.CourseID > 0 {
		enrollment, err := s.repo.GetEnrollment(ctx, assignment.CourseID, doer.ID)
		if err != nil {
			return nil, fmt.Errorf("check enrollment: %w", err)
		}
		if enrollment == nil {
			return nil, fmt.Errorf("you are not enrolled in this course")
		}
	}

	existing, err := s.repo.GetSubmission(ctx, assignment.ID, doer.ID)
	if err != nil {
		return nil, fmt.Errorf("get submission: %w", err)
	}
	if existing != nil {
		return existing, nil
	}

	baseRepo, err := s.forker.GetRepositoryByID(ctx, assignment.RepoID)
	if err != nil {
		return nil, fmt.Errorf("get base repo: %w", err)
	}

	forkName := fmt.Sprintf("%s-%s", doer.Name, baseRepo.Name)

	forkedRepo, err := s.forker.ForkRepositoryAndUpdates(ctx, doer, doer, ForkRepoOptions{
		BaseRepo: baseRepo,
		Name:     forkName,
	})
	if err != nil {
		return nil, fmt.Errorf("fork repo: %w", err)
	}

	submission := &Submission{
		AssignmentID:  assignment.ID,
		UserID:        doer.ID,
		StudentRepoID: forkedRepo.ID,
		Status:        "started",
		CreatedUnix:   time.Now().Unix(),
		UpdatedUnix:   time.Now().Unix(),
	}

	if err := s.repo.CreateSubmission(ctx, submission); err != nil {
		return nil, fmt.Errorf("create submission: %w", err)
	}

	return submission, nil
}
