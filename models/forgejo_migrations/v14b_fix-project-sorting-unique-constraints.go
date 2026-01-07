// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"forgejo.org/modules/setting"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Fix duplicate project sorting values and add unique constraints",
		Upgrade:     fixProjectSortingUniqueConstraints,
	})
}

func fixProjectSortingUniqueConstraints(x *xorm.Engine) error {
	// Step 1: Fix existing duplicates in project_issue (cards)
	// Reassign sequential sorting values within each column
	if err := fixProjectIssueDuplicates(x); err != nil {
		return err
	}

	// Step 2: Fix existing duplicates in project_board (columns)
	// Reassign sequential sorting values within each project
	if err := fixProjectBoardDuplicates(x); err != nil {
		return err
	}

	// Step 3: Add unique constraints
	// Note: Index creation syntax varies by database, but CREATE UNIQUE INDEX is standard
	if _, err := x.Exec("CREATE UNIQUE INDEX UQE_project_issue_column_sorting ON project_issue (project_board_id, sorting)"); err != nil {
		return err
	}
	if _, err := x.Exec("CREATE UNIQUE INDEX UQE_project_board_project_sorting ON project_board (project_id, sorting)"); err != nil {
		return err
	}

	return nil
}

func fixProjectIssueDuplicates(x *xorm.Engine) error {
	switch {
	case setting.Database.Type.IsSQLite3():
		// SQLite: Use UPDATE with subquery
		_, err := x.Exec(`
			UPDATE project_issue SET sorting = (
				SELECT new_sort FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY project_board_id ORDER BY sorting, id) as new_sort
					FROM project_issue
				) ranked WHERE ranked.id = project_issue.id
			)
		`)
		return err

	case setting.Database.Type.IsPostgreSQL():
		// PostgreSQL: Use UPDATE FROM with subquery
		_, err := x.Exec(`
			UPDATE project_issue pi SET sorting = ranked.new_sort
			FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY project_board_id ORDER BY sorting, id) as new_sort
				FROM project_issue
			) ranked
			WHERE pi.id = ranked.id
		`)
		return err

	case setting.Database.Type.IsMySQL():
		// MySQL: Use UPDATE with JOIN
		_, err := x.Exec(`
			UPDATE project_issue pi
			INNER JOIN (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY project_board_id ORDER BY sorting, id) as new_sort
				FROM project_issue
			) ranked ON pi.id = ranked.id
			SET pi.sorting = ranked.new_sort
		`)
		return err

	default:
		// For unknown databases, skip the duplicate fix and try to create the index
		// If there are duplicates, the index creation will fail
		return nil
	}
}

func fixProjectBoardDuplicates(x *xorm.Engine) error {
	switch {
	case setting.Database.Type.IsSQLite3():
		// SQLite: Use UPDATE with subquery
		_, err := x.Exec(`
			UPDATE project_board SET sorting = (
				SELECT new_sort FROM (
					SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY sorting, id) as new_sort
					FROM project_board
				) ranked WHERE ranked.id = project_board.id
			)
		`)
		return err

	case setting.Database.Type.IsPostgreSQL():
		// PostgreSQL: Use UPDATE FROM with subquery
		_, err := x.Exec(`
			UPDATE project_board pb SET sorting = ranked.new_sort
			FROM (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY sorting, id) as new_sort
				FROM project_board
			) ranked
			WHERE pb.id = ranked.id
		`)
		return err

	case setting.Database.Type.IsMySQL():
		// MySQL: Use UPDATE with JOIN
		_, err := x.Exec(`
			UPDATE project_board pb
			INNER JOIN (
				SELECT id, ROW_NUMBER() OVER (PARTITION BY project_id ORDER BY sorting, id) as new_sort
				FROM project_board
			) ranked ON pb.id = ranked.id
			SET pb.sorting = ranked.new_sort
		`)
		return err

	default:
		// For unknown databases, skip the duplicate fix and try to create the index
		// If there are duplicates, the index creation will fail
		return nil
	}
}
