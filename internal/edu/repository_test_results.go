package edu

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
)

func (r *xormRepository) CreateTestResult(ctx context.Context, tr *TestResult) error {
	_, err := db.GetEngine(ctx).Insert(tr)
	if err != nil {
		return fmt.Errorf("insert test result: %w", err)
	}
	return nil
}

func (r *xormRepository) GetTestResultsBySubmission(ctx context.Context, submissionID int64) ([]*TestResult, error) {
	var results []*TestResult
	err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).OrderBy("created_unix DESC").Find(&results)
	if err != nil {
		return nil, fmt.Errorf("find test results: %w", err)
	}
	return results, nil
}

func (r *xormRepository) GetLatestTestResult(ctx context.Context, submissionID int64) (*TestResult, error) {
	tr := &TestResult{}
	has, err := db.GetEngine(ctx).Where("submission_id = ?", submissionID).OrderBy("created_unix DESC").Limit(1).Get(tr)
	if err != nil {
		return nil, fmt.Errorf("get latest test result: %w", err)
	}
	if !has {
		return nil, nil
	}
	return tr, nil
}
