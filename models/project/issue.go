// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package project

import (
	"context"
	"errors"
	"slices"

	"forgejo.org/models/db"
	"forgejo.org/modules/log"
	"forgejo.org/modules/util"
)

// ProjectIssue saves relation from issue to a project
type ProjectIssue struct { //revive:disable-line:exported
	ID        int64 `xorm:"pk autoincr"`
	IssueID   int64 `xorm:"INDEX"`
	ProjectID int64 `xorm:"INDEX"`

	// ProjectColumnID should not be zero since 1.22. If it's zero, the issue will not be displayed on UI and it might result in errors.
	ProjectColumnID int64 `xorm:"'project_board_id' INDEX"`

	// the sorting order on the column
	Sorting int64 `xorm:"NOT NULL DEFAULT 0"`
}

func init() {
	db.RegisterModel(new(ProjectIssue))
}

func deleteProjectIssuesByProjectID(ctx context.Context, projectID int64) error {
	_, err := db.GetEngine(ctx).Where("project_id=?", projectID).Delete(&ProjectIssue{})
	return err
}

// NumClosedIssues return counter of closed issues assigned to a project
func (p *Project) NumClosedIssues(ctx context.Context) int {
	c, err := db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", p.ID, true).
		Cols("issue_id").
		Count()
	if err != nil {
		log.Error("NumClosedIssues: %v", err)
		return 0
	}
	return int(c)
}

// NumOpenIssues return counter of open issues assigned to a project
func (p *Project) NumOpenIssues(ctx context.Context) int {
	c, err := db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", p.ID, false).
		Cols("issue_id").
		Count()
	if err != nil {
		log.Error("NumOpenIssues: %v", err)
		return 0
	}
	return int(c)
}

// MoveIssuesOnProjectColumn moves or keeps issues in a column and sorts them inside that column
func MoveIssuesOnProjectColumn(ctx context.Context, column *Column, sortedIssueIDs map[int64]int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		sess := db.GetEngine(ctx)
		issueIDs := util.ValuesOfMap(sortedIssueIDs)

		// Validate all issues exist and belong to this project
		count, err := sess.Table(new(ProjectIssue)).
			Where("project_id=?", column.ProjectID).
			In("issue_id", issueIDs).Count()
		if err != nil {
			return err
		}
		if int(count) != len(sortedIssueIDs) {
			return errors.New("all issues must belong to the specified project")
		}

		// Build reverse map: issueID → sorting
		sortingByIssue := make(map[int64]int64, len(sortedIssueIDs))
		for sorting, issueID := range sortedIssueIDs {
			sortingByIssue[issueID] = sorting
		}

		// Sort issue IDs to ensure consistent lock ordering across concurrent transactions.
		// This prevents deadlocks when multiple transactions update overlapping rows.
		slices.Sort(issueIDs)

		// Update column assignment and sorting values in consistent order
		for _, issueID := range issueIDs {
			_, err := sess.Exec("UPDATE `project_issue` SET project_board_id=?, sorting=? WHERE issue_id=?",
				column.ID, sortingByIssue[issueID], issueID)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func (c *Column) moveIssuesToAnotherColumn(ctx context.Context, newColumn *Column) error {
	if c.ProjectID != newColumn.ProjectID {
		return errors.New("columns have to be in the same project")
	}

	if c.ID == newColumn.ID {
		return nil
	}

	res := struct {
		MaxSorting int64
		IssueCount int64
	}{}
	if _, err := db.GetEngine(ctx).Select("max(sorting) as max_sorting, count(*) as issue_count").
		Table("project_issue").
		Where("project_id=?", newColumn.ProjectID).
		And("project_board_id=?", newColumn.ID).
		Get(&res); err != nil {
		return err
	}

	issues, err := c.GetIssues(ctx)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		return nil
	}

	nextSorting := util.Iif(res.IssueCount > 0, res.MaxSorting+1, 0)
	return db.WithTx(ctx, func(ctx context.Context) error {
		for i, issue := range issues {
			issue.ProjectColumnID = newColumn.ID
			issue.Sorting = nextSorting + int64(i)
			if _, err := db.GetEngine(ctx).ID(issue.ID).Cols("project_board_id", "sorting").Update(issue); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetProjectCardsInColumn retrieves project issues (cards) for a specific column with pagination
func GetProjectCardsInColumn(ctx context.Context, columnID int64, listOptions db.ListOptions) ([]*ProjectIssue, error) {
	var projectIssues []*ProjectIssue
	sess := db.GetEngine(ctx).Where("project_board_id=?", columnID).OrderBy("sorting, id")

	if listOptions.Page != 0 {
		sess = db.SetSessionPagination(sess, &listOptions)
	}

	return projectIssues, sess.Find(&projectIssues)
}

// GetProjectCard retrieves a specific project card by project and issue ID
func GetProjectCard(ctx context.Context, projectID, issueID int64) (*ProjectIssue, error) {
	projectIssue := &ProjectIssue{}
	has, err := db.GetEngine(ctx).Where("project_id=? AND issue_id=?", projectID, issueID).Get(projectIssue)
	if err != nil {
		return nil, err
	}
	if !has {
		return nil, ErrProjectCardNotExist{ProjectID: projectID, IssueID: issueID}
	}
	return projectIssue, nil
}

// AddIssueToProject adds an issue to a project column with optional sorting position
func AddIssueToProject(ctx context.Context, projectID, issueID, columnID, sorting int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		// Check if issue is already in this project
		exists, err := db.GetEngine(ctx).Where("project_id=? AND issue_id=?", projectID, issueID).Exist(&ProjectIssue{})
		if err != nil {
			return err
		}
		if exists {
			return ErrCardAlreadyInProject{ProjectID: projectID, IssueID: issueID}
		}

		// Validate that the column belongs to the specified project
		column := &Column{}
		has, err := db.GetEngine(ctx).Where("id=? AND project_id=?", columnID, projectID).Get(column)
		if err != nil {
			return err
		}
		if !has {
			return ErrProjectColumnNotExist{ColumnID: columnID}
		}

		// If no sorting position specified, append to end
		if sorting == 0 {
			maxSorting, err := getMaxSortingInColumn(ctx, columnID)
			if err != nil {
				return err
			}
			sorting = maxSorting + 1
		} else {
			// Explicit position specified - shift other cards to make room
			if err := shiftCardsForInsertion(ctx, columnID, sorting, 0); err != nil {
				return err
			}
		}

		projectIssue := &ProjectIssue{
			ProjectID:       projectID,
			IssueID:         issueID,
			ProjectColumnID: columnID,
			Sorting:         sorting,
		}

		_, err = db.GetEngine(ctx).Insert(projectIssue)
		return err
	})
}

// RemoveIssueFromProject removes an issue from a project
func RemoveIssueFromProject(ctx context.Context, projectID, issueID int64) error {
	_, err := db.GetEngine(ctx).Where("project_id=? AND issue_id=?", projectID, issueID).Delete(&ProjectIssue{})
	return err
}

// MoveCardToColumn moves a card to a different column with optional new sorting position
func MoveCardToColumn(ctx context.Context, cardID, newColumnID, newSorting int64) error {
	return db.WithTx(ctx, func(ctx context.Context) error {
		// Get the card first
		card := &ProjectIssue{}
		has, err := db.GetEngine(ctx).ID(cardID).Get(card)
		if err != nil {
			return err
		}
		if !has {
			return ErrProjectCardNotExist{CardID: cardID}
		}

		// Validate that the new column exists and belongs to the same project
		newColumn := &Column{}
		has, err = db.GetEngine(ctx).Where("id=? AND project_id=?", newColumnID, card.ProjectID).Get(newColumn)
		if err != nil {
			return err
		}
		if !has {
			return ErrProjectColumnNotExist{ColumnID: newColumnID}
		}

		// If no new sorting position specified, append to end of new column
		if newSorting == 0 {
			maxSorting, err := getMaxSortingInColumn(ctx, newColumnID)
			if err != nil {
				return err
			}
			newSorting = maxSorting + 1
		} else {
			// Explicit position specified - shift other cards to make room
			// Exclude the card being moved to avoid incrementing itself when moving within the same column
			if err := shiftCardsForInsertion(ctx, newColumnID, newSorting, cardID); err != nil {
				return err
			}
		}

		// Update the card
		card.ProjectColumnID = newColumnID
		card.Sorting = newSorting
		_, err = db.GetEngine(ctx).ID(cardID).Cols("project_board_id", "sorting").Update(card)
		return err
	})
}

// CountCardsInColumn counts the number of cards in a specific column
func CountCardsInColumn(ctx context.Context, columnID int64) (int64, error) {
	return db.GetEngine(ctx).Where("project_board_id=?", columnID).Count(&ProjectIssue{})
}

// CountProjectIssues counts open and closed issues in a project
func CountProjectIssues(ctx context.Context, projectID int64) (open, closed int64, err error) {
	// Count open issues
	open, err = db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", projectID, false).
		Count()
	if err != nil {
		return 0, 0, err
	}

	// Count closed issues
	closed, err = db.GetEngine(ctx).Table("project_issue").
		Join("INNER", "issue", "project_issue.issue_id=issue.id").
		Where("project_issue.project_id=? AND issue.is_closed=?", projectID, true).
		Count()
	if err != nil {
		return 0, 0, err
	}

	return open, closed, nil
}

// getMaxSortingInColumn gets the maximum sorting value in a column
func getMaxSortingInColumn(ctx context.Context, columnID int64) (int64, error) {
	var projectIssue ProjectIssue
	has, err := db.GetEngine(ctx).Where("project_board_id=?", columnID).
		OrderBy("sorting DESC").
		Limit(1).
		Get(&projectIssue)
	if err != nil {
		return 0, err
	}
	if !has {
		return 0, nil
	}
	return projectIssue.Sorting, nil
}

// shiftCardsForInsertion shifts all cards at or after the given position up by 1
// to make room for a new card at that position. The excludeCardID parameter allows
// excluding a specific card (useful when moving within the same column).
func shiftCardsForInsertion(ctx context.Context, columnID, position, excludeCardID int64) error {
	sess := db.GetEngine(ctx).Table("project_issue").
		Where("project_board_id = ? AND sorting >= ?", columnID, position)

	if excludeCardID > 0 {
		sess = sess.And("id != ?", excludeCardID)
	}

	_, err := sess.Incr("sorting", 1).Update(&ProjectIssue{})
	return err
}
