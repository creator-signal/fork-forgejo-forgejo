// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package util

import (
	"context"
	"fmt"

	"forgejo.org/models/issues"
	"forgejo.org/models/organization"
	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

func getF3PathOfRepository(ctx context.Context, repository *repo.Repository) (string, error) {
	if err := repository.LoadOwner(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfRepository LoadOwner: %w", err)
	}
	var ownerPath string
	if repository.Owner.Type == user.UserTypeOrganization {
		ownerPath = f3.ResourceOrganizations
	} else {
		ownerPath = f3.ResourceUsers
	}
	return f3_tree.NewProjectPathString(ownerPath, repository.Owner.Name, repository.Name), nil
}

func getF3PathOfGit(ctx context.Context, repository *repo.Repository) (string, error) {
	if err := repository.LoadOwner(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfRepository LoadOwner: %w", err)
	}
	var kind f3_kind.Kind
	if repository.Owner.Type == user.UserTypeOrganization {
		kind = f3_kind.KindOrganizations
	} else {
		kind = f3_kind.KindUsers
	}
	owners := f3_generic.NewPathFromString("/").SetForge().SetOwners(kind)
	return f3_tree.NewRepositoryPathString(owners.String(), repository.Owner.Name, repository.Name, f3.RepositoryNameDefault), nil
}

func getF3PathOfPullRequest(ctx context.Context, pullRequest *issues.PullRequest) (string, error) {
	if err := pullRequest.LoadIssue(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfPullRequest LoadIssue: %w", err)
	}
	return getF3PathOfIssue(ctx, true, pullRequest.Issue)
}

func getF3PathOfIssue(ctx context.Context, isPull bool, issue *issues.Issue) (string, error) {
	if err := issue.LoadRepo(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfIssue LoadRepo: %w", err)
	}
	projectPath, err := getF3PathOfRepository(ctx, issue.Repo)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfIssue getF3PathOfRepository: %w", err)
	}
	var issuePath string
	if isPull {
		issuePath = f3_tree.NewPullRequestPathString(projectPath, f3_util.ToString(issue.Index))
	} else {
		issuePath = f3_tree.NewIssuePathString(projectPath, f3_util.ToString(issue.Index))
	}
	return issuePath, nil
}

func getF3PathOfComment(ctx context.Context, comment *issues.Comment) (string, error) {
	if err := comment.LoadIssue(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfComment LoadIssue: %w", err)
	}
	commentablePath, err := getF3PathOfIssue(ctx, comment.Issue.IsPull, comment.Issue)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfComment getF3PathOfIssue: %w", err)
	}

	switch comment.Type {
	case issues.CommentTypeComment:
		return f3_tree.NewCommentPathString(commentablePath, f3_util.ToString(comment.ID)), nil
	case issues.CommentTypeReview, issues.CommentTypeCode:
		review, err := issues.GetReviewByID(ctx, comment.ReviewID)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfComment GetReviewByID: %w", err)
		}
		reviewPath, err := getF3PathOfReview(ctx, review)
		if err != nil {
			return "", err
		}
		return f3_tree.NewReviewCommentPathString(reviewPath, f3_util.ToString(comment.ID)), nil
	default:
		err := fmt.Errorf("getF3PathOfComment unexpected comment type %v", comment.Type)
		log.Error(err.Error())
		return "", err
	}
}

func getF3PathOfRelease(ctx context.Context, release *repo.Release) (string, error) {
	if err := release.LoadRepo(ctx); err != nil {
		return "", fmt.Errorf("getF3PathOfRelease LoadRepo: %w", err)
	}
	projectPath, err := getF3PathOfRepository(ctx, release.Repo)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfRelease getF3PathOfRepository: %w", err)
	}
	return f3_tree.NewReleasePathString(projectPath, f3_util.ToString(release.ID)), nil
}

func getF3PathOfAttachment(ctx context.Context, attachment *repo.Attachment) (string, error) {
	var attachablePath string
	if attachment.IssueID != 0 {
		if attachment.CommentID != 0 {
			comment, err := issues.GetCommentByID(ctx, attachment.CommentID)
			if err != nil {
				return "", fmt.Errorf("getF3PathOfAttachment GetCommentByID: %w", err)
			}
			attachablePath, err = getF3PathOfComment(ctx, comment)
			if err != nil {
				return "", fmt.Errorf("getF3PathOfAttachment getF3PathOfComment: %w", err)
			}
		} else {
			issue, err := issues.GetIssueByID(ctx, attachment.IssueID)
			if err != nil {
				return "", fmt.Errorf("getF3PathOfAttachment GetIssueByID: %w", err)
			}
			attachablePath, err = getF3PathOfIssue(ctx, issue.IsPull, issue)
			if err != nil {
				return "", fmt.Errorf("getF3PathOfAttachment getF3PathOfIssue: %w", err)
			}
		}
	} else if attachment.ReleaseID != 0 {
		release, err := repo.GetReleaseByID(ctx, attachment.ReleaseID)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfAttachment GetReleaseByID: %w", err)
		}
		attachablePath, err = getF3PathOfRelease(ctx, release)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfAttachment getF3PathOfRelease: %w", err)
		}
	} else {
		err := fmt.Errorf("getF3PathOfAttachment unexpected attachment with IssueID == 0 and ReleaseID == 0 %+v", attachment)
		log.Error(err.Error())
		return "", err
	}

	return f3_tree.NewAttachmentPathString(attachablePath, f3_util.ToString(attachment.ID)), nil
}

func getF3PathOfReaction(ctx context.Context, reaction *issues.Reaction) (string, error) {
	var reactionablePath string
	if reaction.CommentID != 0 {
		comment, err := issues.GetCommentByID(ctx, reaction.CommentID)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfReaction GetCommentByID: %w", err)
		}
		reactionablePath, err = getF3PathOfComment(ctx, comment)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfReaction getF3PathOfComment: %w", err)
		}
	} else {
		issue, err := issues.GetIssueByID(ctx, reaction.IssueID)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfReaction GetIssueByID: %w", err)
		}
		reactionablePath, err = getF3PathOfIssue(ctx, issue.IsPull, issue)
		if err != nil {
			return "", fmt.Errorf("getF3PathOfReaction getF3PathOfIssue: %w", err)
		}
	}

	return f3_tree.NewReactionPathString(reactionablePath, f3_util.ToString(reaction.ID)), nil
}

func getF3PathOfLabel(ctx context.Context, label *issues.Label) (string, error) {
	repository, err := repo.GetRepositoryByID(ctx, label.RepoID)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfLabel GetRepositoryByID: %w", err)
	}
	projectPath, err := getF3PathOfRepository(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfLabel getF3PathOfRepository: %w", err)
	}
	return f3_tree.NewLabelPathString(projectPath, f3_util.ToString(label.ID)), nil
}

func getF3PathOfMilestone(ctx context.Context, milestone *issues.Milestone) (string, error) {
	repository, err := repo.GetRepositoryByID(ctx, milestone.RepoID)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfMilestone GetRepositoryByID: %w", err)
	}
	projectPath, err := getF3PathOfRepository(ctx, repository)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfMilestone getF3PathOfRepository: %w", err)
	}
	return f3_tree.NewMilestonePathString(projectPath, f3_util.ToString(milestone.ID)), nil
}

func getF3PathOfReview(ctx context.Context, review *issues.Review) (string, error) {
	issue, err := issues.GetIssueByID(ctx, review.IssueID)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfReview GetIssueByID: %w", err)
	}
	if !issue.IsPull {
		return "", fmt.Errorf("getF3PathOfReview the issue %d of review %d must be a pull request", review.ID, issue.ID)
	}
	pullRequestPath, err := getF3PathOfIssue(ctx, issue.IsPull, issue)
	if err != nil {
		return "", fmt.Errorf("getF3PathOfReview getF3PathOfIssue: %w", err)
	}
	return f3_tree.NewReviewPathString(pullRequestPath, f3_util.ToString(review.ID)), nil
}

type GitRepository repo.Repository

func GetF3PathOfResource(ctx context.Context, some any) (string, error) {
	switch o := some.(type) {
	case *repo.Repository:
		return getF3PathOfRepository(ctx, o)
	case *GitRepository:
		return getF3PathOfGit(ctx, (*repo.Repository)(o))
	case *organization.Organization:
		return f3_tree.NewOrganizationPathString(o.Name), nil
	case *user.User:
		return f3_tree.NewUserPathString(o.Name), nil
	case *repo.Topic:
		return f3_tree.NewTopicPathString(f3_util.ToString(o.ID)), nil
	case *issues.Issue:
		return getF3PathOfIssue(ctx, false, o)
	case *issues.PullRequest:
		return getF3PathOfPullRequest(ctx, o)
	case *issues.Comment:
		return getF3PathOfComment(ctx, o)
	case *repo.Release:
		return getF3PathOfRelease(ctx, o)
	case *repo.Attachment:
		return getF3PathOfAttachment(ctx, o)
	case *issues.Reaction:
		return getF3PathOfReaction(ctx, o)
	case *issues.Label:
		return getF3PathOfLabel(ctx, o)
	case *issues.Milestone:
		return getF3PathOfMilestone(ctx, o)
	case *issues.Review:
		return getF3PathOfReview(ctx, o)
	default:
		err := fmt.Errorf("GetF3PathOfResource unexpected type %v %T when converting a Forgejo resource to a F3 path", some, some)
		log.Error(err.Error())
		return "", err
	}
}
