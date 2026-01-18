// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package permissions

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	f3_context "forgejo.org/services/f3/context"
	permissions_comment "forgejo.org/services/permissions/comment"
	permissions_context "forgejo.org/services/permissions/context"
	permissions_tests "forgejo.org/services/permissions/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestPermissionsPermissionsCommentGet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	for _, testCase := range []struct {
		name    string
		fixture func(t *testing.T) (repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment)
		run     func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) permissions_context.CheckFunc
		error   string
	}{
		{
			name: "Get",
			fixture: func(t *testing.T) (repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) {
				t.Helper()
				comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{}, builder.Eq{"type": issues_model.CommentTypeComment})
				issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
				repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: issue.RepoID})
				return repo, issue, comment
			},
			run: func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) permissions_context.CheckFunc {
				return permissions_comment.Get(repo.ID, issue.ID, comment)
			},
		},
		{
			name: "Get unrelated ID",
			fixture: func(t *testing.T) (repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) {
				t.Helper()
				comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{}, builder.Eq{"type": issues_model.CommentTypeComment})
				goodIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: comment.IssueID})
				repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: goodIssue.RepoID})

				badIssue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 10})
				require.NotEqual(t, goodIssue.ID, badIssue.ID)
				issue = badIssue
				return repo, issue, comment
			},
			run: func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) permissions_context.CheckFunc {
				return permissions_comment.Get(repo.ID, issue.ID, comment)
			},
			error: "NotFound: comment",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repo, issue, comment := testCase.fixture(t)

			buildCtx := func() permissions_context.PermissionsContext {
				return f3_context.Get(f3_context.WithF3(t.Context()))
			}
			if testCase.error == "" {
				permissions_tests.ComplianceRepositoryGet(t, buildCtx, repo, testCase.run(repo, issue, comment))
			} else {
				ctx := buildCtx()

				doer := unittest.AssertExistsAndLoadBean(t, &user_model.User{IsAdmin: true})
				scope, err := auth_model.AccessTokenScope("all").Normalize()
				require.NoError(t, err)
				accessToken := &auth_model.AccessToken{
					UID:              doer.ID,
					Name:             testCase.name,
					Scope:            scope,
					ResourceAllRepos: true,
				}
				require.NoError(t, auth_model.NewAccessToken(t.Context(), accessToken))
				permissions_context.SetToken(ctx, accessToken.Token)

				require.NoError(t, permissions_context.SetRepository(ctx, repo.ID))
				assert.ErrorContains(t, testCase.run(repo, issue, comment)(ctx), testCase.error)
			}
		})
	}
}
