// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package e2e

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/db"
	git_model "forgejo.org/models/git"
	issues_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/indexer/stats"
	"forgejo.org/modules/optional"
	api "forgejo.org/modules/structs"
	"forgejo.org/modules/timeutil"
	issue_service "forgejo.org/services/issue"
	pull_service "forgejo.org/services/pull"
	commitstatus_service "forgejo.org/services/repository/commitstatus"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/services/wiki"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/xorm/convert"
)

// first entry represents filename
// the following entries define the full file content over time
type FileChanges struct {
	Filename  string
	CommitMsg string
	Versions  []string
}

// performs additional repo setup as needed
type SetupRepo func(*user_model.User, *repo_model.Repository)

// put your Git repo declarations in here
// feel free to amend the helper function below or use the raw variant directly
func DeclareGitRepos(t *testing.T) func() {
	now := timeutil.TimeStampNow()
	postIssue := func(repo *repo_model.Repository, user *user_model.User, age int64, title, content string) {
		issue := &issues_model.Issue{
			RepoID:      repo.ID,
			PosterID:    user.ID,
			Poster:      user,
			Title:       title,
			Content:     content,
			CreatedUnix: now.Add(-age),
		}
		require.NoError(t, issue_service.NewIssue(db.DefaultContext, repo, issue, nil, nil, nil))
	}

	// Helpers for creating test data in the dependency board e2e repo.
	postIssueWithMilestone := func(repo *repo_model.Repository, user *user_model.User, age int64, title, content string, milestoneID int64) *issues_model.Issue {
		issue := &issues_model.Issue{
			RepoID:      repo.ID,
			PosterID:    user.ID,
			Poster:      user,
			Title:       title,
			Content:     content,
			MilestoneID: milestoneID,
			CreatedUnix: now.Add(-age),
		}
		require.NoError(t, issue_service.NewIssue(db.DefaultContext, repo, issue, nil, nil, nil))
		return issue
	}

	createMilestone := func(repo *repo_model.Repository, name string) int64 {
		m := &issues_model.Milestone{
			RepoID: repo.ID,
			Name:   name,
		}
		require.NoError(t, issues_model.NewMilestone(db.DefaultContext, m))
		return m.ID
	}

	addDependency := func(user *user_model.User, blocked, blocker *issues_model.Issue) {
		require.NoError(t, issues_model.CreateIssueDependency(db.DefaultContext, user, blocked, blocker))
	}

	createBranchWithChange := func(repo *repo_model.Repository, user *user_model.User, oldBranch, newBranch, filename, content string) string {
		resp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user, &files_service.ChangeRepoFilesOptions{
			Files: []*files_service.ChangeRepoFile{{
				Operation:     "update",
				TreePath:      filename,
				ContentReader: strings.NewReader(content),
			}},
			Message:   fmt.Sprintf("change %s on %s", filename, newBranch),
			OldBranch: oldBranch,
			NewBranch: newBranch,
			Author: &files_service.IdentityOptions{
				Name:  user.Name,
				Email: user.Email,
			},
			Committer: &files_service.IdentityOptions{
				Name:  user.Name,
				Email: user.Email,
			},
			Dates: &files_service.CommitDateOptions{
				Author:    time.Now(),
				Committer: time.Now(),
			},
		})
		require.NoError(t, err)
		return resp.Commit.SHA
	}

	createPR := func(repo *repo_model.Repository, user *user_model.User, branch, title string) *issues_model.PullRequest {
		pullIssue := &issues_model.Issue{
			RepoID:      repo.ID,
			PosterID:    user.ID,
			Poster:      user,
			Title:       title,
			IsPull:      true,
			CreatedUnix: now.Add(-50),
		}
		pr := &issues_model.PullRequest{
			HeadRepoID: repo.ID,
			BaseRepoID: repo.ID,
			HeadBranch: branch,
			BaseBranch: "main",
			HeadRepo:   repo,
			BaseRepo:   repo,
			Type:       issues_model.PullRequestGitea,
		}
		require.NoError(t, pull_service.NewPullRequest(git.DefaultContext, repo, pullIssue, nil, nil, pr, nil))
		return pr
	}

	setCommitStatus := func(repo *repo_model.Repository, user *user_model.User, sha string, state api.CommitStatusState, context string) {
		require.NoError(t, commitstatus_service.CreateCommitStatus(db.DefaultContext, repo, user, sha, &git_model.CommitStatus{
			State:     state,
			TargetURL: "https://example.com/ci",
			Context:   context,
		}))
	}

	cleanupFunctions := []func(){
		newRepo(t, 2, "diff-test", nil, []FileChanges{{
			Filename: "testfile",
			Versions: []string{"hello", "hallo", "hola", "native", "ubuntu-latest", "- runs-on: ubuntu-latest", "- runs-on: debian-latest"},
		}}, nil),
		newRepo(t, 2, "language-stats-test", nil, []FileChanges{{
			Filename: "main.rs",
			Versions: []string{"fn main() {", "println!(\"Hello World!\");", "}"},
		}}, nil),
		newRepo(t, 2, "mentions-highlighted", nil, []FileChanges{
			{
				Filename:  "history1.md",
				Versions:  []string{""},
				CommitMsg: "A commit message which mentions @user2 in the title\nand has some additional text which mentions @user1",
			},
			{
				Filename:  "history2.md",
				Versions:  []string{""},
				CommitMsg: "Another commit which mentions @user1 in the title\nand @user2 in the text",
			},
		}, nil),
		newRepo(t, 2, "file-uploads", nil, []FileChanges{{
			Filename: "UPLOAD_TEST.md",
			Versions: []string{"# File upload test\nUse this repo to test various file upload features in new branches."},
		}}, nil),
		newRepo(t, 2, "unicode-escaping", &tests.DeclarativeRepoOptions{
			EnabledUnits: optional.Some([]unit_model.Type{unit_model.TypeCode, unit_model.TypeWiki}),
		}, []FileChanges{{
			Filename: "a-file",
			Versions: []string{"{a}{а}"},
		}}, func(user *user_model.User, repo *repo_model.Repository) {
			wiki.InitWiki(db.DefaultContext, repo)
			wiki.AddWikiPage(db.DefaultContext, user, repo, "Home", "{a}{а}", "{a}{а}")
			wiki.AddWikiPage(db.DefaultContext, user, repo, "_Sidebar", "{a}{а}", "{a}{а}")
			wiki.AddWikiPage(db.DefaultContext, user, repo, "_Footer", "{a}{а}", "{a}{а}")
		}),
		newRepo(t, 2, "multiple-combo-boxes", nil, []FileChanges{{
			Filename: ".forgejo/issue_template/multi-combo-boxes.yaml",
			Versions: []string{`
name: "Multiple combo-boxes"
description: "To show something"
body:
- type: textarea
  id: textarea-one
  attributes:
    label: one
- type: textarea
  id: textarea-two
  attributes:
    label: two
`},
		}}, nil),
		newRepo(t, 11, "dependency-test", &tests.DeclarativeRepoOptions{
			UnitConfig: optional.Some(map[unit_model.Type]convert.Conversion{
				unit_model.TypeIssues: &repo_model.IssuesConfig{
					EnableDependencies: true,
				},
			}),
		}, []FileChanges{}, func(user *user_model.User, repo *repo_model.Repository) {
			postIssue(repo, user, 500, "first issue here", "an issue created earlier")
			postIssue(repo, user, 400, "second issue here (not 1)", "not the right issue, but in the right repo")
			postIssue(repo, user, 300, "third issue here", "depends on things")
			postIssue(repo, user, 200, "unrelated issue", "shrug emoji")
			postIssue(repo, user, 100, "newest issue", "very new")
		}),
		newRepo(t, 11, "dependency-test-2", &tests.DeclarativeRepoOptions{
			UnitConfig: optional.Some(map[unit_model.Type]convert.Conversion{
				unit_model.TypeIssues: &repo_model.IssuesConfig{
					EnableDependencies: true,
				},
			}),
		}, []FileChanges{}, func(user *user_model.User, repo *repo_model.Repository) {
			postIssue(repo, user, 450, "right issue", "an issue containing word right")
			postIssue(repo, user, 150, "left issue", "an issue containing word left")
		}),
		// Declarative repo for dependency board e2e tests: 5 issues with a dependency
		// chain and two milestones, enabling tests for column layout, highlighting,
		// filtering, pane editing, and dependency creation.
		newRepo(t, 11, "dependency-board-test", &tests.DeclarativeRepoOptions{
			UnitConfig: optional.Some(map[unit_model.Type]convert.Conversion{
				unit_model.TypeIssues: &repo_model.IssuesConfig{
					EnableDependencies: true,
				},
				unit_model.TypePullRequests: &repo_model.PullRequestsConfig{
					AllowMerge: true,
				},
			}),
			EnabledUnits: optional.Some([]unit_model.Type{
				unit_model.TypeCode,
				unit_model.TypeIssues,
				unit_model.TypePullRequests,
			}),
		}, []FileChanges{}, func(user *user_model.User, repo *repo_model.Repository) {
			v1 := createMilestone(repo, "v1.0")
			v2 := createMilestone(repo, "v2.0")

			issueA := postIssueWithMilestone(repo, user, 500, "setup database", "initialize the database", v1)
			issueB := postIssueWithMilestone(repo, user, 400, "build API", "create the API layer", v1)
			issueC := postIssueWithMilestone(repo, user, 300, "add tests", "write test suite", v1)
			issueD := postIssueWithMilestone(repo, user, 200, "write docs", "document the project", v2)
			issueE := postIssueWithMilestone(repo, user, 100, "deploy", "deploy to production", v1)

			addDependency(user, issueB, issueA)
			addDependency(user, issueC, issueB)
			addDependency(user, issueC, issueD)
			addDependency(user, issueE, issueC)

			prOpenSHA := createBranchWithChange(repo, user, "main", "pr-open-branch", "README.md", "# Open PR repo\nUpdated by open PR.")
			prOpen := createPR(repo, user, "pr-open-branch", "implement feature X")
			_ = prOpen
			setCommitStatus(repo, user, prOpenSHA, api.CommitStatusSuccess, "ci/test")

			prDraftSHA := createBranchWithChange(repo, user, "main", "pr-draft-branch", "README.md", "# Draft PR repo\nWork in progress.")
			prDraft := createPR(repo, user, "pr-draft-branch", "WIP: refactor database layer")
			_ = prDraft
			setCommitStatus(repo, user, prDraftSHA, api.CommitStatusPending, "ci/test")

			prMergedSHA := createBranchWithChange(repo, user, "main", "pr-merged-branch", "README.md", "# Merged PR repo\nFinal content.")
			prMerged := createPR(repo, user, "pr-merged-branch", "fix login bug")
			gitRepo, err := git.OpenRepository(git.DefaultContext, repo.RepoPath())
			require.NoError(t, err)
			require.NoError(t, pull_service.Merge(git.DefaultContext, prMerged, user, gitRepo, repo_model.MergeStyleMerge, prMerged.HeadCommitID, "merge PR", false))
			gitRepo.Close()
			setCommitStatus(repo, user, prMergedSHA, api.CommitStatusSuccess, "ci/test")

			prClosedSHA := createBranchWithChange(repo, user, "main", "pr-closed-branch", "README.md", "# Closed PR repo\nRejected content.")
			prClosed := createPR(repo, user, "pr-closed-branch", "remove deprecated API")
			require.NoError(t, issue_service.ChangeStatus(db.DefaultContext, prClosed.Issue, user, "", true))
			setCommitStatus(repo, user, prClosedSHA, api.CommitStatusFailure, "ci/test")
		}),
		newRepo(t, 2, "long-diff-test", nil, []FileChanges{{
			Filename: "test-README.md",
			Versions: []string{
				readStringFile(t, "tests/e2e/declarative-repo/long-diff-test/0-README.md"),
			},
		}}, func(user *user_model.User, repo *repo_model.Repository) {
			commit1Sha := addCommitToBranch(t, user, repo, "main", "test-branch", "test-README.md", "",
				readStringFile(t, "tests/e2e/declarative-repo/long-diff-test/1-README.md"))
			commit2Sha := addCommitToBranch(t, user, repo, "test-branch", "test-branch", "test-README.md", commit1Sha,
				readStringFile(t, "tests/e2e/declarative-repo/long-diff-test/2-README.md"))
			addCommitToBranch(t, user, repo, "test-branch", "test-branch", "test-README.md", commit2Sha,
				readStringFile(t, "tests/e2e/declarative-repo/long-diff-test/3-README.md"))
		}),
		newRepo(t, 2, "huge-diff-test", nil, []FileChanges{{
			Filename: "glossary.po",
			Versions: []string{
				func() string {
					var sb strings.Builder
					sb.Write([]byte("0"))
					for i := 1; i < 2000; i++ {
						sb.WriteString(strconv.Itoa(i))
						sb.WriteByte('\n')
					}
					return sb.String()
				}(),
			},
		}}, func(user *user_model.User, repo *repo_model.Repository) {
			addCommitToBranch(t, user, repo, "main", "main-2", "glossary.po", "",
				func() string {
					var sb strings.Builder
					sb.Write([]byte("0"))
					for i := 1; i < 2000; i++ {
						sb.WriteString(strconv.Itoa(i))
						if i%12 == 0 {
							sb.WriteString("Blub")
						}
						sb.WriteByte('\n')
					}
					return sb.String()
				}())
		}),
		// add your repo declarations here
	}

	return func() {
		for _, cleanup := range cleanupFunctions {
			cleanup()
		}
	}
}

func readStringFile(t *testing.T, fn string) string {
	c, err := os.ReadFile(fn)
	require.NoError(t, err)
	return string(c)
}

func newRepo(t *testing.T, userID int64, repoName string, initOpts *tests.DeclarativeRepoOptions, fileChanges []FileChanges, setup SetupRepo) func() {
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: userID})

	opts := tests.DeclarativeRepoOptions{}
	if initOpts != nil {
		opts = *initOpts
	}
	opts.Name = optional.Some(repoName)
	if !opts.EnabledUnits.Has() {
		opts.EnabledUnits = optional.Some([]unit_model.Type{unit_model.TypeCode, unit_model.TypeIssues})
	}
	somerepo, _, cleanupFunc := tests.CreateDeclarativeRepoWithOptions(t, user, opts)

	var lastCommitID string
	for _, file := range fileChanges {
		for i, version := range file.Versions {
			operation := "update"
			if i == 0 {
				operation = "create"
			}

			// default to unique commit messages
			commitMsg := file.CommitMsg
			if commitMsg == "" {
				commitMsg = fmt.Sprintf("Patch: %s-%d", file.Filename, i+1)
			}

			resp, err := files_service.ChangeRepoFiles(git.DefaultContext, somerepo, user, &files_service.ChangeRepoFilesOptions{
				Files: []*files_service.ChangeRepoFile{{
					Operation:     operation,
					TreePath:      file.Filename,
					ContentReader: strings.NewReader(version),
				}},
				Message:   commitMsg,
				OldBranch: "main",
				NewBranch: "main",
				Author: &files_service.IdentityOptions{
					Name:  user.Name,
					Email: user.Email,
				},
				Committer: &files_service.IdentityOptions{
					Name:  user.Name,
					Email: user.Email,
				},
				Dates: &files_service.CommitDateOptions{
					Author:    time.Now(),
					Committer: time.Now(),
				},
				LastCommitID: lastCommitID,
			})
			require.NoError(t, err)
			assert.NotEmpty(t, resp)

			lastCommitID = resp.Commit.SHA
		}
	}

	if setup != nil {
		setup(user, somerepo)
	}

	err := stats.UpdateRepoIndexer(somerepo)
	require.NoError(t, err)

	return cleanupFunc
}

func addCommitToBranch(t *testing.T, user *user_model.User, repo *repo_model.Repository, oldBranch, newBranch, filename, lastSha, content string) string {
	resp, err := files_service.ChangeRepoFiles(git.DefaultContext, repo, user, &files_service.ChangeRepoFilesOptions{
		Files: []*files_service.ChangeRepoFile{{
			Operation:     "update",
			TreePath:      filename,
			ContentReader: strings.NewReader(content),
		}},
		Message:   "add commit to branch",
		OldBranch: oldBranch,
		NewBranch: newBranch,
		Author: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Committer: &files_service.IdentityOptions{
			Name:  user.Name,
			Email: user.Email,
		},
		Dates: &files_service.CommitDateOptions{
			Author:    time.Now(),
			Committer: time.Now(),
		},
		LastCommitID: lastSha,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, resp)
	return resp.Commit.SHA
}
