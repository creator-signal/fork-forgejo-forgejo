package edu

import (
	"context"
)

func (s *service) GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error) {
	return s.repo.GetAssignments(ctx, repoID)
}
