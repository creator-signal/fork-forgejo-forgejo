package edu

import (
	"context"
	"fmt"
)

func (s *service) GetTestResults(ctx context.Context, submissionID int64) ([]*TestResult, error) {
	return s.repo.GetTestResultsBySubmission(ctx, submissionID)
}

func (s *service) GetLatestTestResult(ctx context.Context, submissionID int64) (*TestResult, error) {
	return s.repo.GetLatestTestResult(ctx, submissionID)
}

func (s *service) GradeSubmission(ctx context.Context, submissionID int64, grade int, comment string, gradedByID int64) error {
	if grade < 0 || grade > 100 {
		return fmt.Errorf("grade must be between 0 and 100")
	}
	return s.repo.GradeSubmission(ctx, submissionID, grade, comment, gradedByID)
}
