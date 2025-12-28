// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package issues_test

import (
	"testing"

	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/require"
)

func TestIssueUpdateParent(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	parentIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 23})
	subIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 24})

	err := subIssue.UpdateParentIssue(db.DefaultContext, parentIssue, user1)
	require.NoError(t, err)

	unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 23}, "parent_id IS NULL")
	unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 24, ParentIssueID: &parentIssue.ID})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:               issues_model.CommentTypeAddSubIssue,
		PosterID:           user1.ID,
		IssueID:            parentIssue.ID,
		ParentOrSubIssueID: subIssue.ID,
	})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:               issues_model.CommentTypeAddParentIssue,
		PosterID:           user1.ID,
		IssueID:            subIssue.ID,
		ParentOrSubIssueID: parentIssue.ID,
	})

	err = subIssue.UpdateParentIssue(db.DefaultContext, nil, user1)
	require.NoError(t, err)

	unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 23, ParentIssueID: nil})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 24, ParentIssueID: nil})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:               issues_model.CommentTypeRemoveSubIssue,
		PosterID:           user1.ID,
		IssueID:            parentIssue.ID,
		ParentOrSubIssueID: subIssue.ID,
	})
	unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{
		Type:               issues_model.CommentTypeRemoveParentIssue,
		PosterID:           user1.ID,
		IssueID:            subIssue.ID,
		ParentOrSubIssueID: parentIssue.ID,
	})
}

func TestSubIssueNoCircular(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user1 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
	rootIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 23})

	issue1 := testCreateIssue(t, 63, 1, "Test sub-issue", "issue content", false)
	err := issue1.UpdateParentIssue(db.DefaultContext, rootIssue, user1)
	require.NoError(t, err)

	err = rootIssue.UpdateParentIssue(db.DefaultContext, issue1, user1)
	require.EqualError(t, err, issues_model.ErrCircularParentIssue{rootIssue.ID, issue1.ID}.Error())
}
