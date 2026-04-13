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

func (s *service) ResetToAutoGrade(ctx context.Context, submissionID int64) error {
	latestResult, err := s.repo.GetLatestTestResult(ctx, submissionID)
	if err != nil {
		return fmt.Errorf("get latest test result: %w", err)
	}

	grade := -1
	if latestResult != nil {
		grade = latestResult.Score
	}

	return s.repo.ResetToAutoGrade(ctx, submissionID, grade)
}
