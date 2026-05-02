package edu

import (
	"context"
	"fmt"
	"time"
)

func (s *service) GetAssignmentsForUser(ctx context.Context, userID int64) ([]*Assignment, error) {
	return s.repo.GetAssignmentsForUser(ctx, userID)
}

func (s *service) UpdateAssignment(ctx context.Context, a *Assignment) error {
	if a.Title == "" {
		return fmt.Errorf("title is required")
	}
	a.UpdatedUnix = time.Now().Unix()
	return s.repo.UpdateAssignment(ctx, a)
}

func (s *service) DeleteAssignment(ctx context.Context, id int64) error {
	return s.repo.DeleteAssignment(ctx, id)
}
