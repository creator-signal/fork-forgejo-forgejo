// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package comment

import (
	"testing"

	auth_model "forgejo.org/models/auth"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	f3_context "forgejo.org/services/f3/context"
	f3_permissions "forgejo.org/services/f3/permissions"
	f3_permissions_tests "forgejo.org/services/f3/permissions/tests"

	f3_assert "code.forgejo.org/f3/gof3/v3/util/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

func TestPermissionsPermissionsCommentGet(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	for _, testCase := range []struct {
		name    string
		fixture func(t *testing.T) (repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment)
		run     func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) f3_permissions.CheckFunc
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
			run: func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) f3_permissions.CheckFunc {
				return Get(repo.ID, issue.ID, comment)
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
			run: func(repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) f3_permissions.CheckFunc {
				return Get(repo.ID, issue.ID, comment)
			},
			error: "NotFound: comment",
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repo, issue, comment := testCase.fixture(t)

			buildCtx := func() *f3_context.F3 {
				return f3_context.Get(f3_context.WithF3(t.Context(), f3_context.New()))
			}
			if testCase.error == "" {
				f3_permissions_tests.CompliancePublicRepositoryGet(t, buildCtx, repo, testCase.run(repo, issue, comment))
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
				f3_permissions.SetToken(ctx, accessToken.Token)

				f3_permissions.SetRepositoryFromID(ctx, repo.ID)
				f3_assert.PanicErrorContains(t, func() { testCase.run(repo, issue, comment)(ctx) }, testCase.error)
			}
		})
	}
}
