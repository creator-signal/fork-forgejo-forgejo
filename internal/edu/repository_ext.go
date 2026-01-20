package edu

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) GetAssignments(ctx context.Context, repoID int64) ([]*Assignment, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "repo_id", "title", "description", "deadline_unix", "created_unix", "updated_unix").
		From("edu_assignments").
		Where(sq.Eq{"repo_id": repoID}).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var assignments []*Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.RepoID, &a.Title, &a.Description, &a.DeadlineUnix, &a.CreatedUnix, &a.UpdatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		assignments = append(assignments, &a)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return assignments, nil
}
