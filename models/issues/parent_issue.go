// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
)

// LoadParentIssue load parent issue of this issue.
func (issue *Issue) LoadParentIssue(ctx context.Context) (err error) {
	if issue.ParentIssueID != nil && !issue.isParentIssueLoaded && (issue.ParentIssue == nil || issue.ParentIssue.ID != *issue.ParentIssueID) {
		issue.ParentIssue, err = GetIssueByID(ctx, *issue.ParentIssueID)
		if err != nil {
			return err
		}

		issue.isParentIssueLoaded = true
	}

	return nil
}

// GetSubIssuesByIssueID returns all sub-issues that belong to given issue by ID.
func GetSubIssuesByIssueID(ctx context.Context, issueID int64) ([]*Issue, error) {
	var subIssues []*Issue
	return subIssues, db.GetEngine(ctx).
		Table("issue").
		Where("issue.parent_id = ?", issueID).
		Asc("issue.created_unix").
		Find(&subIssues)
}

// LoadSubIssues load sub-issues of this issue.
func (issue *Issue) LoadSubIssues(ctx context.Context) (err error) {
	if !issue.isSubIssuesLoaded {
		issue.SubIssues, err = GetSubIssuesByIssueID(ctx, issue.ID)
		if err != nil {
			return err
		}
		issue.isSubIssuesLoaded = true
	}
	return nil
}

func CountSubIssues(ctx context.Context, issueID int64) (int64, error) {
	return db.GetEngine(ctx).
		Table("issue").
		Where("issue.parent_id = ?", issueID).
		Count()
}

// ErrSubIssuesTooMany represents a "SubIssuesTooMany" kind of error.
type ErrSubIssuesTooMany struct {
	ID       int64
	ParentID int64
	RootID   int64
}

// IsErrSubIssuesTooMany checks if an error is a ErrSubIssuesTooMany.
func IsErrSubIssuesTooMany(err error) bool {
	_, ok := err.(ErrSubIssuesTooMany)
	return ok
}

func (err ErrSubIssuesTooMany) Error() string {
	return fmt.Sprintf("sub-issues count has reached limit [id: %d, parent: %d, root: %d]", err.ID, err.ParentID, err.RootID)
}

func (err ErrSubIssuesTooMany) Unwrap() error {
	return util.ErrTooMany
}

// ErrSubIssuesTooDeep represents a "SubIssuesTooDeep" kind of error.
type ErrSubIssuesTooDeep struct {
	ID       int64
	ParentID int64
	RootID   int64
}

// IsErrSubIssuesTooDeep checks if an error is a ErrSubIssuesTooDeep.
func IsErrSubIssuesTooDeep(err error) bool {
	_, ok := err.(ErrSubIssuesTooDeep)
	return ok
}

func (err ErrSubIssuesTooDeep) Error() string {
	return fmt.Sprintf("sub-issues depth has reached limit [id: %d, parent: %d, root: %d]", err.ID, err.ParentID, err.RootID)
}

// ErrCircularParentIssue represents a "CircularParentIssue" kind of error.
type ErrCircularParentIssue struct {
	ID       int64
	ParentID int64
}

// IsErrCircularParentIssue checks if an error is a ErrCircularParentIssue.
func IsErrCircularParentIssue(err error) bool {
	_, ok := err.(ErrCircularParentIssue)
	return ok
}

func (err ErrCircularParentIssue) Error() string {
	return fmt.Sprintf("circular parent issues [id: %d, parent: %d]", err.ID, err.ParentID)
}

// LookupRootIssue resolves the root issue of this issue, which has no parent issue
func (issue *Issue) LookupRootIssue(ctx context.Context) (root *Issue, depth int, err error) {
	root = issue
	depth = 0
	for {
		if root.ParentIssueID == nil {
			return root, depth, nil
		}
		if err = root.LoadParentIssue(ctx); err != nil {
			return nil, 0, err
		}
		root = root.ParentIssue
		depth++

		// Enforce maximum allowed sub-issue depth to avoid excessive resource usage.
		// If depth exceeds configured maximum, return a specific error so callers
		// can handle it (for example, when trying to link a new parent).
		if setting.Repository.Issue.MaxSubIssuesDepth > 0 && depth > setting.Repository.Issue.MaxSubIssuesDepth {
			// Use the current parent as ParentID; RootID is set to the same parent here
			// because the actual top root may not be known at this point in traversal.
			return nil, 0, ErrSubIssuesTooDeep{
				ID:       issue.ID,
				ParentID: root.ID,
				RootID:   root.ID,
			}
		}
	}
}

// CountSubIssues counts count of all sub-issues of this issue recursively
func (issue *Issue) CountSubIssues(ctx context.Context) (int, error) {
	var count int64
	_, err := db.GetEngine(ctx).SQL(`
		WITH RECURSIVE sub_issues AS (
			SELECT id FROM issue WHERE parent_id = ?
			UNION ALL
			SELECT issue.id FROM issue
			INNER JOIN sub_issues ON issue.parent_id = sub_issues.id
		)
		SELECT count(*) FROM sub_issues
	`, issue.ID).Get(&count)
	return int(count), err
}

// UpdateParentIssue adds issue to another issue as a sub-issue.
// Setting parent issue to nil means removing parent issue
func (issue *Issue) UpdateParentIssue(ctx context.Context, parent *Issue, doer *user_model.User) (err error) {
	if issue.ParentIssueID != nil && parent != nil && *issue.ParentIssueID == parent.ID {
		return nil
	}
	if issue.ParentIssueID == nil && parent == nil {
		return nil
	}
	if err = issue.LoadParentIssue(ctx); err != nil {
		return err
	}
	oldParent := issue.ParentIssue

	if parent != nil {
		root, depth, err := parent.LookupRootIssue(ctx)
		if err != nil {
			return err
		}

		// Validate count and depth limitation
		if depth+1 > setting.Repository.Issue.MaxSubIssuesDepth {
			return ErrSubIssuesTooDeep{issue.ID, parent.ID, root.ID}
		}
		rootCount, err := root.CountSubIssues(ctx)
		if err != nil {
			return err
		}
		if rootCount+1 > setting.Repository.Issue.MaxSubIssues {
			return ErrSubIssuesTooMany{issue.ID, parent.ID, root.ID}
		}

		// Validate no circular parent issues
		curParent := parent
		// after parent.LookupRootIssue, all parent issues on the path to root has been loaded
		for curParent != nil {
			if curParent.ID == issue.ID {
				return ErrCircularParentIssue{issue.ID, parent.ID}
			}
			curParent = curParent.ParentIssue
		}
	}

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return err
	}
	defer committer.Close()

	var parentID *int64
	if parent != nil {
		parentID = &parent.ID
	}

	// Update parent ID
	if err = UpdateIssueCols(ctx, &Issue{ID: issue.ID, ParentIssueID: parentID}, "parent_id"); err != nil {
		return err
	}

	issue.ParentIssueID = parentID
	// invalidates caches
	issue.isParentIssueLoaded = false
	if parent != nil {
		parent.isSubIssuesLoaded = false
	}

	// Make the comment
	if oldParent != nil {
		// removed old parent
		if err = createSubIssueComment(ctx, doer, oldParent, issue, false); err != nil {
			return fmt.Errorf("createSubIssueComment, unlink old: %w", err)
		}
	}
	if parent != nil {
		// added new parent
		if err = createSubIssueComment(ctx, doer, parent, issue, true); err != nil {
			return fmt.Errorf("createSubIssueComment, link new: %w", err)
		}
	}

	return committer.Commit()
}

// GetTotalSubIssues returns the number of sub-issues
func (issue *Issue) GetTotalSubIssues() int {
	return len(issue.SubIssues)
}

// GetClosedSubIssues returns the number of closed sub-issues
func (issue *Issue) GetClosedSubIssues() int {
	count := 0
	for _, sub := range issue.SubIssues {
		if sub.IsClosed {
			count++
		}
	}
	return count
}

// GetSubIssuesProgress returns the progress of sub-issues in percentage
func (issue *Issue) GetSubIssuesProgress() int {
	total := issue.GetTotalSubIssues()
	if total == 0 {
		return 0
	}
	return (issue.GetClosedSubIssues() * 100) / total
}

func (issue *Issue) HasSubIssues() bool {
	ctx := context.Background()
	count, err := CountSubIssues(ctx, issue.ID)
	if err != nil {
		return false
	}

	return count > 0
}

func (issue *Issue) SubIssuesCount() int64 {
	ctx := context.Background()
	count, err := CountSubIssues(ctx, issue.ID)
	if err != nil {
		return 0
	}

	return count
}
