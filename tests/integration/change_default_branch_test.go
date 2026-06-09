// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	actions_module "forgejo.org/modules/actions"
	"forgejo.org/modules/gitrepo"
	"forgejo.org/modules/optional"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"
	"forgejo.org/tests/forgery"
	"github.com/stretchr/testify/require"
	"net/url"
	"strings"
)

func TestChangeDefaultBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repo.OwnerID})

	session := loginUser(t, owner.Name)
	branchesURL := fmt.Sprintf("/%s/%s/settings/branches", owner.Name, repo.Name)

	req := NewRequestWithValues(t, "POST", branchesURL, map[string]string{
		"action": "default_branch",
		"branch": "DefaultBranch",
	})
	session.MakeRequest(t, req, http.StatusSeeOther)

	req = NewRequestWithValues(t, "POST", branchesURL, map[string]string{
		"action": "default_branch",
		"branch": "does_not_exist",
	})
	session.MakeRequest(t, req, http.StatusNotFound)
}

func TestChangeDefaultBranchUpdatesSchedules(t *testing.T) {

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	//create opts for the Repo

	type expectedSpec struct {
		cron     string
		timeZone optional.Option[string]
	}

	testWorkflow := []struct {
		name                  string
		workflowID            string
		workflowDirectory     string
		workflowContent       string
		expectedWorkflowTitle string
		expectedCronSpecs     []expectedSpec
	}{
		{
			name:              "GitHub",
			workflowID:        "scheduled.yml",
			workflowDirectory: ".forgejo/workflows",
			workflowContent: `
on:
  schedule:
    - cron: "30 5,17 * * *"
jobs:
  test:
    steps:
      - run: echo OK
`,
			expectedWorkflowTitle: ".forgejo/workflows/scheduled.yml",
			expectedCronSpecs:     []expectedSpec{{cron: "30 5,17 * * *", timeZone: optional.None[string]()}},
		},
	}

	onApplicationRun(t, func(t *testing.T, u *url.URL) {

		var sha string
		repo := forgery.CreateRepository(t, user, &forgery.CreateRepositoryOptions{
			Files: forgery.MapFS{
				fmt.Sprintf("%s/%s", testWorkflow[0].workflowDirectory, testWorkflow[0].workflowID): forgery.MapFile(testWorkflow[0].workflowContent),
			},
			LatestSha: &sha,
		})
		forgery.EnableRepoUnits(t, repo, unit.TypeActions)

		schedules, err := db.Find[actions_model.ActionSchedule](t.Context(), actions_model.FindScheduleOptions{RepoID: repo.ID})
		require.NoError(t, err)
		require.Len(t, schedules, 1)

		t.Logf("default branch before: %s", repo.DefaultBranch)

		require.True(t, repo.UnitEnabled(t.Context(), unit.TypeActions))

		t.Logf("path of the repo: %s", repo.RepoPath())

		gitRepo, err := gitrepo.OpenRepository(t.Context(), repo)
		require.NoError(t, err)
		defer gitRepo.Close()

		//create branch
		err = repo_service.CreateNewBranch(t.Context(), user, repo, gitRepo, repo.DefaultBranch, "test")
		require.NoError(t, err)

		err = repo_service.SetRepoDefaultBranch(t.Context(), repo, gitRepo, "test")
		require.NoError(t, err)

		enabled := repo.UnitEnabled(t.Context(), unit.TypeActions)
		require.True(t, enabled)

		t.Logf("Actions unit: %t", enabled)
		commit, err := gitRepo.GetBranchCommit(repo.DefaultBranch)
		require.NoError(t, err)

		branch, err := commit.GetBranchName()
		require.NoError(t, err)
		t.Logf("Branch name: %s from commit", branch)

		workflows, err := actions_module.DetectScheduledWorkflows(gitRepo, commit)
		require.NoError(t, err)

		t.Logf("detected workflows: %d", len(workflows))
		t.Logf("detected workflow: %s", workflows[0].EntryName)
		t.Logf("default branch after: %s", repo.DefaultBranch)

		// Implement check schedules
		schedules, err = db.Find[actions_model.ActionSchedule](t.Context(), actions_model.FindScheduleOptions{RepoID: repo.ID})

		t.Logf("Schedules count from test branch: %d", len(schedules))
		require.NoError(t, err)
		require.Len(t, schedules, 0)

		err = repo_service.SetRepoDefaultBranch(t.Context(), repo, gitRepo, "main")
		require.NoError(t, err)

		schedules, err = db.Find[actions_model.ActionSchedule](t.Context(), actions_model.FindScheduleOptions{RepoID: repo.ID})

		t.Logf("Schedules count from main branch: %d", len(schedules))
		require.NoError(t, err)
		require.Len(t, schedules, 1)

		commit, err = gitRepo.GetBranchCommit("test")
		require.NoError(t, err)

		_, err = files_service.ChangeRepoFiles(
			t.Context(),
			repo,
			user,
			&files_service.ChangeRepoFilesOptions{
				LastCommitID: commit.ID.String(),
				OldBranch:    "test",
				NewBranch:    "test",
				Message:      "update workflow",
				Files: []*files_service.ChangeRepoFile{
					{
						Operation: "update",
						TreePath:  testWorkflow[0].expectedWorkflowTitle,
						ContentReader: strings.NewReader(`
on:
  schedule:
    - cron: "0 * * * *"
jobs:
  test:
    steps:
      - run: echo updated
`),
					},
				},
			},
		)
		require.NoError(t, err)

		err = repo_service.SetRepoDefaultBranch(t.Context(), repo, gitRepo, "test")
		require.NoError(t, err)

		commit, err = gitRepo.GetBranchCommit(repo.DefaultBranch)
		require.NoError(t, err)

		branch, err = commit.GetBranchName()
		require.NoError(t, err)
		t.Logf("Branch name: %s from commit", branch)

		schedules, err = db.Find[actions_model.ActionSchedule](t.Context(), actions_model.FindScheduleOptions{RepoID: repo.ID})

		require.NoError(t, err)
		require.Len(t, schedules, 1)

		//TODO
		// - Create a push on test branch
		// - Assrt all the schedules
		// - chek the ref to test on ActionSchedule
		// - chek the ref to main on ActionSchedule
	})
}
