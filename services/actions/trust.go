// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"fmt"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	issues_model "forgejo.org/models/issues"
	access_model "forgejo.org/models/perm/access"
	unit_model "forgejo.org/models/unit"
	actions_module "forgejo.org/modules/actions"
	"forgejo.org/modules/log"
)

type UserTrust string

const (
	UserTrustDenied   = UserTrust("deny")
	UserAlwaysTrusted = UserTrust("always")
	UserTrustedOnce   = UserTrust("once")
	UserTrustRevoked  = UserTrust("revoke")
)

func CleanupActionUser(ctx context.Context) error {
	return actions_model.RevokeInactiveActionUser(ctx)
}

func loadPullRequestAttributes(ctx context.Context, pr *issues_model.PullRequest) error {
	if err := pr.LoadIssue(ctx); err != nil {
		return err
	}

	if err := pr.Issue.LoadRepo(ctx); err != nil {
		return err
	}

	return pr.Issue.LoadPoster(ctx)
}

// cancels or approves runs and keep track of posters that are to always be trusted
func UpdateTrustedWithPullRequest(ctx context.Context, doerID int64, pr *issues_model.PullRequest, trusted UserTrust) error {
	if err := updateTrusted(ctx, pr, trusted); err != nil {
		return err
	}

	switch trusted {
	case UserAlwaysTrusted, UserTrustedOnce:
		return PullRequestApprove(ctx, doerID, pr)
	case UserTrustRevoked, UserTrustDenied:
		return PullRequestCancel(ctx, pr)
	default:
		return fmt.Errorf("UpdateTrustedWithPullRequest: unknown trust %v", trusted)
	}
}

func updateTrusted(ctx context.Context, pr *issues_model.PullRequest, trusted UserTrust) error {
	switch trusted {
	case UserTrustedOnce:
		return nil
	case UserAlwaysTrusted:
		if err := loadPullRequestAttributes(ctx, pr); err != nil {
			return err
		}
		return actions_model.InsertActionUser(ctx, &actions_model.ActionUser{
			UserID:                  pr.Issue.Poster.ID,
			RepoID:                  pr.Issue.Repo.ID,
			TrustedWithPullRequests: true,
		})
	case UserTrustDenied:
		return nil
	case UserTrustRevoked:
		if err := loadPullRequestAttributes(ctx, pr); err != nil {
			return err
		}
		return actions_model.DeleteActionUserByUserIDAndRepoID(ctx, pr.Issue.Poster.ID, pr.Issue.Repo.ID)
	default:
		return fmt.Errorf("updateTrusted: unknown trust %v", trusted)
	}
}

func SetRunTrustForPullRequest(ctx context.Context, run *actions_model.ActionRun, pr *issues_model.PullRequest) error {
	if pr == nil {
		return nil
	}

	if err := pr.LoadIssue(ctx); err != nil {
		return err
	}

	run.IsForkPullRequest = pr.IsForkPullRequest()
	run.PullRequestPosterID = pr.Issue.PosterID
	run.PullRequestID = pr.ID

	needApproval, err := ifNeedApproval(ctx, run, pr)
	if err != nil {
		return err
	}

	if needApproval {
		run.NeedApproval = needApproval
	}

	return nil
}

func ifNeedApproval(ctx context.Context, run *actions_model.ActionRun, pr *issues_model.PullRequest) (bool, error) {
	// 1. don't need approval if it's not a fork PR
	// 2. don't need approval if the event is `pull_request_target` since the workflow will run in the context of base branch
	// 		see https://docs.github.com/en/actions/managing-workflow-runs/approving-workflow-runs-from-public-forks#about-workflow-runs-from-public-forks
	if !run.IsForkPullRequest || run.TriggerEvent == actions_module.GithubEventPullRequestTarget {
		return false, nil
	}

	trusted, err := GetPullRequestPosterIsTrustedWithActions(ctx, pr)
	if err != nil {
		return false, err
	}

	return trusted == PosterIsNotTrustedWithActions, nil
}

type PosterTrust string

const (
	PosterIsNotTrustedWithActions        = PosterTrust("no")
	PosterIsExplicitlyTrustedWithActions = PosterTrust("explicitly")
	PosterIsImplicitlyTrustedWithActions = PosterTrust("implicitly")
)

func GetPullRequestPosterIsTrustedWithActions(ctx context.Context, pr *issues_model.PullRequest) (PosterTrust, error) {
	if err := loadPullRequestAttributes(ctx, pr); err != nil {
		return "", err
	}

	return posterIsTrustedWithPullRequest(ctx, pr)
}

func posterIsTrustedWithPullRequest(ctx context.Context, pr *issues_model.PullRequest) (PosterTrust, error) {
	implicitlyTrusted, err := posterIsImplicitlyTrustedWithPullRequest(ctx, pr)
	if err != nil {
		return "", err
	}
	if implicitlyTrusted {
		log.Trace("%s is implicitly trusted to run actions in repository %s", pr.Issue.Poster, pr.Issue.Repo)
		return PosterIsImplicitlyTrustedWithActions, nil
	}

	explicitlyTrusted, err := posterIsExplicitlyTrustedWithPullRequest(ctx, pr)
	if err != nil {
		return "", err
	}
	if explicitlyTrusted {
		log.Trace("%s is explicitly trusted to run actions in repository %s", pr.Issue.Poster, pr.Issue.Repo)
		return PosterIsExplicitlyTrustedWithActions, nil
	}

	log.Trace("%s is not trusted to run actions in repository %s", pr.Issue.Poster, pr.Issue.Repo)
	return PosterIsNotTrustedWithActions, nil
}

func posterIsImplicitlyTrustedWithPullRequest(ctx context.Context, pr *issues_model.PullRequest) (bool, error) {
	// users that are trusted to create a pull request that is not from a fork
	// are also implicitly trusted to run workflows
	if !pr.IsForkPullRequest() {
		log.Trace("a pull request that is not from a fork nor AGit is implicitly trusted to run actions")
		return true, nil
	}

	// users with write permission to the actions unit are trusted to
	// run actions
	permission, err := access_model.GetUserRepoPermission(ctx, pr.Issue.Repo, pr.Issue.Poster)
	if err != nil {
		return false, err
	}
	if permission.CanWrite(unit_model.TypeActions) {
		log.Trace("%s is a member of a team with write permissions to the Action unit on %s", pr.Issue.Poster, pr.Issue.Repo)
		return true, nil
	}

	return false, nil
}

func posterIsExplicitlyTrustedWithPullRequest(ctx context.Context, pr *issues_model.PullRequest) (bool, error) {
	// there is no need to check if the user is blocked because it is not
	// allowed to create a pull request
	if pr.Issue.Poster.IsRestricted {
		log.Trace("%v is restricted and cannot be trusted with pull requests", pr.Issue.Poster)
		return false, nil
	}

	user, err := actions_model.GetActionUserByUserIDAndRepoIDAndUpdateAccess(ctx, pr.Issue.Poster.ID, pr.Issue.Repo.ID)
	if err != nil {
		log.Trace("%v is not explicitly trusted with pull requests on repository %v", pr.Issue.Poster, pr.Issue.Repo)
		if actions_model.IsErrUserNotExist(err) {
			return false, nil
		}
		return false, err
	}

	log.Trace("%v is explicitly trusted with pull requests on repository %v", pr.Issue.Poster, pr.Issue.Repo)
	return user.TrustedWithPullRequests, nil
}

func RevokeTrustByRepoIDAndPosterID(ctx context.Context, repoID, posterID int64) error {
	if err := actions_model.DeleteActionUserByUserIDAndRepoID(ctx, posterID, repoID); err != nil {
		return err
	}

	runs, err := actions_model.GetRunsNotDoneByRepoIDAndPullRequestPosterID(ctx, repoID, posterID)
	if err != nil {
		return err
	}

	for _, run := range runs {
		if err := CancelRun(ctx, run); err != nil {
			return err
		}
	}
	return nil
}

func PullRequestCancel(ctx context.Context, pr *issues_model.PullRequest) error {
	runs, err := actions_model.GetRunsNotDoneByRepoIDAndPullRequestID(ctx, pr.Issue.RepoID, pr.ID)
	if err != nil {
		return err
	}

	for _, run := range runs {
		if err := CancelRun(ctx, run); err != nil {
			return err
		}
	}
	return nil
}

func PullRequestApprove(ctx context.Context, doerID int64, pr *issues_model.PullRequest) error {
	runs, err := actions_model.GetRunsThatNeedApproval(ctx, pr.Issue.RepoID, pr.ID)
	if err != nil {
		return err
	}

	for _, run := range runs {
		if err := db.WithTx(ctx, func(ctx context.Context) error {
			jobs, err := actions_model.GetRunJobsByRunID(ctx, run.ID)
			if err != nil {
				return err
			}
			for _, job := range jobs {
				if len(job.Needs) == 0 && job.Status.IsBlocked() {
					job.Status = actions_model.StatusWaiting
					_, err := UpdateRunJob(ctx, job, nil, "status")
					if err != nil {
						return err
					}
				}
			}
			CreateCommitStatus(ctx, jobs...)

			return actions_model.UpdateRunApprovalByID(ctx, run.ID, actions_model.DoesNotNeedApproval, doerID)
		}); err != nil {
			return err
		}
	}
	return nil
}
