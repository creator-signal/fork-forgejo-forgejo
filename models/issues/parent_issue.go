// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
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

// LoadSubIssueRepos loads repositories for sub-issues of this issue.
func (issue *Issue) LoadSubIssueRepos(ctx context.Context) error {
	if len(issue.SubIssues) == 0 {
		return nil
	}
	issueList := IssueList(issue.SubIssues)
	_, err := issueList.LoadRepositories(ctx)
	return err
}

// CountSubIssues counts count of all sub-issues of this issue
func CountSubIssues(ctx context.Context, issueID int64) (int64, error) {
	return db.GetEngine(ctx).
		Table("issue").
		Where("issue.parent_id = ?", issueID).
		Count()
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
	}
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
		if _, _, err := parent.LookupRootIssue(ctx); err != nil {
			return err
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
		if err = RemoveIssueDependencyNoComment(ctx, doer, oldParent, issue, DependencyTypeBlockedBy); err != nil && !IsErrDependencyNotExists(err) {
			return fmt.Errorf("RemoveIssueDependencyNoComment: %w", err)
		}

		// removed old parent
		if err = createSubIssueComment(ctx, doer, oldParent, issue, false); err != nil {
			return fmt.Errorf("createSubIssueComment, unlink old: %w", err)
		}
	}
	if parent != nil {
		if err = CreateIssueDependencyNoComment(ctx, doer, parent, issue); err != nil && !IsErrDependencyExists(err) {
			return fmt.Errorf("CreateIssueDependencyNoComment: %w", err)
		}

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
