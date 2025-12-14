// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package util

import (
	"fmt"
	"testing"

	issues_model "forgejo.org/models/issues"
	org_model "forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"

	f3_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertConvert[T any](t *testing.T, some T, expected string) {
	t.Helper()
	actual, err := ConvertForgejoToF3Path(t.Context(), some)
	require.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestConvertForgejoToF3Path(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	t.Run("user", func(t *testing.T) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		assertConvert(t, user, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").String())
	})
	t.Run("organization", func(t *testing.T) {
		organization := unittest.AssertExistsAndLoadBean(t, &org_model.Organization{ID: 3})
		assertConvert(t, organization, f3_generic.NewPathFromString("/").SetForge().SetOrganizations().SetOrganization("org3").String())
	})
	t.Run("topic", func(t *testing.T) {
		topic := unittest.AssertExistsAndLoadBean(t, &repo_model.Topic{ID: 2})
		assertConvert(t, topic, f3_generic.NewPathFromString("/").SetForge().SetTopics().SetTopic("2").String())
	})
	t.Run("organization project", func(t *testing.T) {
		project := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3})
		assertConvert(t, project, f3_generic.NewPathFromString("/").SetForge().SetOrganizations().SetOrganization("org3").SetProjects().SetProject("repo3").String())
	})
	t.Run("user project", func(t *testing.T) {
		project := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})
		assertConvert(t, project, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo2").String())
	})
	t.Run("issue", func(t *testing.T) {
		issue := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2})
		assertConvert(t, issue, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("2").String())
	})
	t.Run("pull_request", func(t *testing.T) {
		pullRequest := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		assertConvert(t, pullRequest, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetPullRequests().SetPullRequest("3").String()) // Index of PR with ID 2 is 3
	})
	t.Run("issue comment", func(t *testing.T) {
		comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2})
		assertConvert(t, comment, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("1").SetComments().SetComment("2").String())
	})
	t.Run("pull_request comment", func(t *testing.T) {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1})
		pullRequest := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2})
		pullRequest.LoadIssue(t.Context())
		pullRequest.Issue.LoadRepo(t.Context())
		comment, err := issues_model.CreateComment(t.Context(), &issues_model.CreateCommentOptions{
			Type:  issues_model.CommentTypeComment,
			Doer:  user,
			Issue: pullRequest.Issue,
			Repo:  pullRequest.Issue.Repo,
		})
		require.NoError(t, err)
		assertConvert(t, comment, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetPullRequests().SetPullRequest("3").SetComments().SetComment(fmt.Sprintf("%d", comment.ID)).String())
	})
	t.Run("release", func(t *testing.T) {
		release := unittest.AssertExistsAndLoadBean(t, &repo_model.Release{ID: 1})
		assertConvert(t, release, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetReleases().SetRelease("1").String())
	})
	t.Run("issue attachment", func(t *testing.T) {
		attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 1})
		assertConvert(t, attachment, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("1").SetAttachments().SetAttachment("1").String())
	})
	t.Run("release attachment", func(t *testing.T) {
		attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 9})
		assertConvert(t, attachment, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetReleases().SetRelease("1").SetAttachments().SetAttachment("9").String())
	})
	t.Run("comment attachment", func(t *testing.T) {
		attachment := unittest.AssertExistsAndLoadBean(t, &repo_model.Attachment{ID: 7})
		assertConvert(t, attachment, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("1").SetComments().SetComment("2").SetAttachments().SetAttachment("7").String())
	})
	t.Run("issue reaction", func(t *testing.T) {
		reaction := unittest.AssertExistsAndLoadBean(t, &issues_model.Reaction{ID: 1})
		assertConvert(t, reaction, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("1").SetReactions().SetReaction("1").String())
	})
	t.Run("comment reaction", func(t *testing.T) {
		reaction := unittest.AssertExistsAndLoadBean(t, &issues_model.Reaction{ID: 4})
		assertConvert(t, reaction, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetIssues().SetIssue("1").SetComments().SetComment("2").SetReactions().SetReaction("4").String())
	})
	t.Run("label", func(t *testing.T) {
		label := unittest.AssertExistsAndLoadBean(t, &issues_model.Label{ID: 1})
		assertConvert(t, label, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetLabels().SetLabel("1").String())
	})
	t.Run("milestone", func(t *testing.T) {
		milestone := unittest.AssertExistsAndLoadBean(t, &issues_model.Milestone{ID: 1})
		assertConvert(t, milestone, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetMilestones().SetMilestone("1").String())
	})
	t.Run("review", func(t *testing.T) {
		review := unittest.AssertExistsAndLoadBean(t, &issues_model.Review{ID: 1})
		assertConvert(t, review, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetPullRequests().SetPullRequest("2").SetReviews().SetReview("1").String())
	})
	t.Run("reviewcomment", func(t *testing.T) {
		commentReview := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 9})
		assertConvert(t, commentReview, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetPullRequests().SetPullRequest("2").SetReviews().SetReview("20").SetReviewComments().SetReviewComment("9").String())
		commentReviewCode := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 7})
		assertConvert(t, commentReviewCode, f3_generic.NewPathFromString("/").SetForge().SetUsers().SetUser("user2").SetProjects().SetProject("repo1").SetPullRequests().SetPullRequest("3").SetReviews().SetReview("10").SetReviewComments().SetReviewComment("7").String())
	})
}
