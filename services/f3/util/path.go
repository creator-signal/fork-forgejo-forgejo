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
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

func convertForgejoRepositoryToF3Path(ctx context.Context, repository *repo.Repository) (string, error) {
	if err := repository.LoadOwner(ctx); err != nil {
		return "", fmt.Errorf("LoadOwner: %w", err)
	}
	var ownerPath string
	if repository.Owner.Type == user.UserTypeOrganization {
		ownerPath = f3.ResourceOrganizations
	} else {
		ownerPath = f3.ResourceUsers
	}
	return f3_tree.NewProjectPathString(ownerPath, repository.Owner.Name, repository.Name), nil
}

func convertForgejoPullRequestToF3Path(ctx context.Context, pullRequest *issues.PullRequest) (string, error) {
	if err := pullRequest.LoadIssue(ctx); err != nil {
		return "", fmt.Errorf("LoadIssue: %w", err)
	}
	return convertForgejoIssueToF3Path(ctx, true, pullRequest.Issue)
}

func convertForgejoIssueToF3Path(ctx context.Context, isPull bool, issue *issues.Issue) (string, error) {
	if err := issue.LoadRepo(ctx); err != nil {
		return "", fmt.Errorf("LoadRepo: %w", err)
	}
	projectPath, err := convertForgejoRepositoryToF3Path(ctx, issue.Repo)
	if err != nil {
		return "", err
	}
	var issuePath string
	if isPull {
		issuePath = f3_tree.NewPullRequestPathString(projectPath, f3_util.ToString(issue.Index))
	} else {
		issuePath = f3_tree.NewIssuePathString(projectPath, f3_util.ToString(issue.Index))
	}
	return issuePath, nil
}

func convertForgejoCommentToF3Path(ctx context.Context, comment *issues.Comment) (string, error) {
	if err := comment.LoadIssue(ctx); err != nil {
		return "", fmt.Errorf("LoadIssue: %w", err)
	}
	commentablePath, err := convertForgejoIssueToF3Path(ctx, comment.Issue.IsPull, comment.Issue)
	if err != nil {
		return "", err
	}

	switch comment.Type {
	case issues.CommentTypeComment:
		return f3_tree.NewCommentPathString(commentablePath, f3_util.ToString(comment.ID)), nil
	case issues.CommentTypeReview, issues.CommentTypeCode:
		review, err := issues.GetReviewByID(ctx, comment.ReviewID)
		if err != nil {
			return "", fmt.Errorf("GetReviewByID: %w", err)
		}
		reviewPath, err := convertForgejoReviewToF3Path(ctx, review)
		if err != nil {
			return "", err
		}
		return f3_tree.NewReviewCommentPathString(reviewPath, f3_util.ToString(comment.ID)), nil
	default:
		err := fmt.Errorf("unexpected comment type %v", comment.Type)
		log.Error(err.Error())
		return "", err
	}
}

func convertForgejoReleaseToF3Path(ctx context.Context, release *repo.Release) (string, error) {
	if err := release.LoadRepo(ctx); err != nil {
		return "", fmt.Errorf("LoadRepo: %w", err)
	}
	projectPath, err := convertForgejoRepositoryToF3Path(ctx, release.Repo)
	if err != nil {
		return "", err
	}
	return f3_tree.NewReleasePathString(projectPath, f3_util.ToString(release.ID)), nil
}

func convertForgejoAttachmentToF3Path(ctx context.Context, attachment *repo.Attachment) (string, error) {
	var attachablePath string
	if attachment.IssueID != 0 {
		if attachment.CommentID != 0 {
			comment, err := issues.GetCommentByID(ctx, attachment.CommentID)
			if err != nil {
				return "", fmt.Errorf("GetCommentByID: %w", err)
			}
			attachablePath, err = convertForgejoCommentToF3Path(ctx, comment)
			if err != nil {
				return "", err
			}
		} else {
			issue, err := issues.GetIssueByID(ctx, attachment.IssueID)
			if err != nil {
				return "", fmt.Errorf("GetIssueByID: %w", err)
			}
			attachablePath, err = convertForgejoIssueToF3Path(ctx, issue.IsPull, issue)
			if err != nil {
				return "", err
			}
		}
	} else if attachment.ReleaseID != 0 {
		release, err := repo.GetReleaseByID(ctx, attachment.ReleaseID)
		if err != nil {
			return "", fmt.Errorf("GetReleaseByID: %w", err)
		}
		attachablePath, err = convertForgejoReleaseToF3Path(ctx, release)
		if err != nil {
			return "", err
		}
	} else {
		err := fmt.Errorf("unexpected attachment with IssueID == 0 and ReleaseID == 0 %+v", attachment)
		log.Error(err.Error())
		return "", err
	}

	return f3_tree.NewAttachmentPathString(attachablePath, f3_util.ToString(attachment.ID)), nil
}

func convertForgejoReactionToF3Path(ctx context.Context, reaction *issues.Reaction) (string, error) {
	var reactionablePath string
	if reaction.CommentID != 0 {
		comment, err := issues.GetCommentByID(ctx, reaction.CommentID)
		if err != nil {
			return "", fmt.Errorf("GetCommentByID: %w", err)
		}
		reactionablePath, err = convertForgejoCommentToF3Path(ctx, comment)
		if err != nil {
			return "", err
		}
	} else {
		issue, err := issues.GetIssueByID(ctx, reaction.IssueID)
		if err != nil {
			return "", fmt.Errorf("GetIssueByID: %w", err)
		}
		reactionablePath, err = convertForgejoIssueToF3Path(ctx, issue.IsPull, issue)
		if err != nil {
			return "", err
		}
	}

	return f3_tree.NewReactionPathString(reactionablePath, f3_util.ToString(reaction.ID)), nil
}

func convertForgejoLabelToF3Path(ctx context.Context, label *issues.Label) (string, error) {
	repository, err := repo.GetRepositoryByID(ctx, label.RepoID)
	if err != nil {
		return "", fmt.Errorf("GetRepositoryByID: %w", err)
	}
	projectPath, err := convertForgejoRepositoryToF3Path(ctx, repository)
	if err != nil {
		return "", err
	}
	return f3_tree.NewLabelPathString(projectPath, f3_util.ToString(label.ID)), nil
}

func convertForgejoMilestoneToF3Path(ctx context.Context, milestone *issues.Milestone) (string, error) {
	repository, err := repo.GetRepositoryByID(ctx, milestone.RepoID)
	if err != nil {
		return "", fmt.Errorf("GetRepositoryByID: %w", err)
	}
	projectPath, err := convertForgejoRepositoryToF3Path(ctx, repository)
	if err != nil {
		return "", err
	}
	return f3_tree.NewMilestonePathString(projectPath, f3_util.ToString(milestone.ID)), nil
}

func convertForgejoReviewToF3Path(ctx context.Context, review *issues.Review) (string, error) {
	issue, err := issues.GetIssueByID(ctx, review.IssueID)
	if err != nil {
		return "", fmt.Errorf("GetIssueByID: %w", err)
	}
	if !issue.IsPull {
		return "", fmt.Errorf("the issue %d of review %d must be a pull request", review.ID, issue.ID)
	}
	pullRequestPath, err := convertForgejoIssueToF3Path(ctx, issue.IsPull, issue)
	if err != nil {
		return "", err
	}
	return f3_tree.NewReviewPathString(pullRequestPath, f3_util.ToString(review.ID)), nil
}

func ConvertForgejoToF3Path(ctx context.Context, some any) (string, error) {
	switch o := some.(type) {
	case *repo.Repository:
		return convertForgejoRepositoryToF3Path(ctx, o)
	case *organization.Organization:
		return f3_tree.NewOrganizationPathString(o.Name), nil
	case *user.User:
		return f3_tree.NewUserPathString(o.Name), nil
	case *repo.Topic:
		return f3_tree.NewTopicPathString(f3_util.ToString(o.ID)), nil
	case *issues.Issue:
		return convertForgejoIssueToF3Path(ctx, false, o)
	case *issues.PullRequest:
		return convertForgejoPullRequestToF3Path(ctx, o)
	case *issues.Comment:
		return convertForgejoCommentToF3Path(ctx, o)
	case *repo.Release:
		return convertForgejoReleaseToF3Path(ctx, o)
	case *repo.Attachment:
		return convertForgejoAttachmentToF3Path(ctx, o)
	case *issues.Reaction:
		return convertForgejoReactionToF3Path(ctx, o)
	case *issues.Label:
		return convertForgejoLabelToF3Path(ctx, o)
	case *issues.Milestone:
		return convertForgejoMilestoneToF3Path(ctx, o)
	case *issues.Review:
		return convertForgejoReviewToF3Path(ctx, o)
	default:
		err := fmt.Errorf("unexpected type %v %T", some, some)
		log.Error(err.Error())
		return "", err
	}
}
