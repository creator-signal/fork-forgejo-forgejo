package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) CreateTestResult(ctx context.Context, tr *TestResult) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_test_results").
		Columns("submission_id", "commit_sha", "score", "details", "created_unix").
		Values(tr.SubmissionID, tr.CommitSHA, tr.Score, tr.Details, tr.CreatedUnix).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&tr.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetTestResultsBySubmission(ctx context.Context, submissionID int64) ([]*TestResult, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "submission_id", "commit_sha", "score", "details", "created_unix").
		From("edu_test_results").
		Where(sq.Eq{"submission_id": submissionID}).
		OrderBy("created_unix DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var results []*TestResult
	for rows.Next() {
		var tr TestResult
		if err := rows.Scan(&tr.ID, &tr.SubmissionID, &tr.CommitSHA, &tr.Score, &tr.Details, &tr.CreatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		results = append(results, &tr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return results, nil
}

func (r *dbRepository) GetLatestTestResult(ctx context.Context, submissionID int64) (*TestResult, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "submission_id", "commit_sha", "score", "details", "created_unix").
		From("edu_test_results").
		Where(sq.Eq{"submission_id": submissionID}).
		OrderBy("created_unix DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var tr TestResult
	err = row.Scan(&tr.ID, &tr.SubmissionID, &tr.CommitSHA, &tr.Score, &tr.Details, &tr.CreatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &tr, nil
}
