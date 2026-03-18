package edu

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

func (r *dbRepository) CreateImportDraft(ctx context.Context, d *ImportDraft) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Insert("edu_import_draft").
		Columns("course_id", "creator_id", "status", "raw_csv", "created_unix").
		Values(d.CourseID, d.CreatorID, d.Status, d.RawCSV, d.CreatedUnix).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	err = r.runner.QueryRowContext(ctx, query, args...).Scan(&d.ID)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) GetImportDraft(ctx context.Context, id int64) (*ImportDraft, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "course_id", "creator_id", "status", "raw_csv", "created_unix").
		From("edu_import_draft").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	row := r.runner.QueryRowContext(ctx, query, args...)

	var d ImportDraft
	err = row.Scan(&d.ID, &d.CourseID, &d.CreatorID, &d.Status, &d.RawCSV, &d.CreatedUnix)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan row: %w", err)
	}

	return &d, nil
}

func (r *dbRepository) CreateImportDraftRows(ctx context.Context, rows []*ImportDraftRow) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	for _, row := range rows {
		query, args, err := psql.Insert("edu_import_draft_row").
			Columns("draft_id", "full_name", "email", "group_name", "username", "role", "status", "error_msg", "created_unix").
			Values(row.DraftID, row.FullName, row.Email, row.Group, row.Username, row.Role, row.Status, row.ErrorMsg, row.CreatedUnix).
			Suffix("RETURNING id").
			ToSql()
		if err != nil {
			return fmt.Errorf("build query: %w", err)
		}

		err = r.runner.QueryRowContext(ctx, query, args...).Scan(&row.ID)
		if err != nil {
			return fmt.Errorf("exec query for row %q: %w", row.FullName, err)
		}
	}

	return nil
}

func (r *dbRepository) GetImportDraftRows(ctx context.Context, draftID int64) ([]*ImportDraftRow, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Select("id", "draft_id", "full_name", "email", "group_name", "username", "role", "status", "error_msg", "created_unix").
		From("edu_import_draft_row").
		Where(sq.Eq{"draft_id": draftID}).
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.runner.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("exec query: %w", err)
	}
	defer rows.Close()

	var result []*ImportDraftRow
	for rows.Next() {
		var row ImportDraftRow
		if err := rows.Scan(&row.ID, &row.DraftID, &row.FullName, &row.Email, &row.Group, &row.Username, &row.Role, &row.Status, &row.ErrorMsg, &row.CreatedUnix); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		result = append(result, &row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return result, nil
}

func (r *dbRepository) UpdateImportDraftRow(ctx context.Context, row *ImportDraftRow) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("edu_import_draft_row").
		Set("username", row.Username).
		Set("email", row.Email).
		Set("status", row.Status).
		Set("error_msg", row.ErrorMsg).
		Where(sq.Eq{"id": row.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) UpdateImportDraft(ctx context.Context, d *ImportDraft) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query, args, err := psql.Update("edu_import_draft").
		Set("status", d.Status).
		Where(sq.Eq{"id": d.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("exec query: %w", err)
	}

	return nil
}

func (r *dbRepository) DeleteImportDraft(ctx context.Context, id int64) error {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// Delete rows first
	query, args, err := psql.Delete("edu_import_draft_row").
		Where(sq.Eq{"draft_id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete rows query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete rows: %w", err)
	}

	// Delete draft
	query, args, err = psql.Delete("edu_import_draft").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete draft query: %w", err)
	}

	_, err = r.runner.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete draft: %w", err)
	}

	return nil
}
