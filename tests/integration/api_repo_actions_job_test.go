// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIGetActionJob(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)

		// Create a repo with an action workflow
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-job-test-repo",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/test.yml",
					ContentReader: strings.NewReader(`name: test
on: push
jobs:
  job1:
    runs-on: docker
    steps:
      - name: Step 1
        run: echo "Hello from job1 step1"
      - name: Step 2
        run: echo "Hello from job1 step2"
  job2:
    runs-on: docker
    needs: job1
    steps:
      - name: Step 1
        run: echo "Hello from job2"
`),
				},
			},
		)
		defer f()

		// Wait for the workflow to be created
		assert.Eventually(t, func() bool {
			count := unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID})
			return count == 1
		}, 1*time.Minute, 1*time.Second, "workflow run should be created within 1 minute")
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})

		// Get jobs for this run
		jobs, err := actions_model.GetRunJobsByRunID(t.Context(), run.ID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(jobs), 1)

		// Test getting a specific job
		req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0",
			repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)

		var jobResp map[string]any
		DecodeJSON(t, resp, &jobResp)

		// Verify response structure
		assert.NotNil(t, jobResp["id"])
		assert.NotNil(t, jobResp["run_id"])
		assert.NotNil(t, jobResp["name"])
		assert.NotNil(t, jobResp["status"])
		assert.NotNil(t, jobResp["run"])

		// Verify run information is included
		runInfo := jobResp["run"].(map[string]any)
		assert.InDelta(t, float64(run.ID), runInfo["id"], 0.01)
		assert.NotEmpty(t, runInfo["title"])
		assert.NotEmpty(t, runInfo["status"])

		// Test non-existent job index
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/999",
			repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)

		// Test non-existent run
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/9999/jobs/0",
			repo.OwnerName, repo.Name).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})
}

func TestAPIGetActionJobLogs(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)

		// Create a repo with a simple workflow
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-job-logs-test-repo",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/simple.yml",
					ContentReader: strings.NewReader(`name: simple
on: push
jobs:
  test:
    runs-on: docker
    steps:
      - name: Test Step
        run: |
          echo "Line 1 of output"
          echo "Line 2 of output"
          echo "Line 3 of output"
`),
				},
			},
		)
		defer f()

		// Wait for the workflow to be created
		assert.Eventually(t, func() bool {
			count := unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID})
			return count >= 1
		}, 1*time.Minute, 1*time.Second, "workflow run should be created within 1 minute")

		// Get the run that was created
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})

		// Get jobs for this run
		jobs, err := actions_model.GetRunJobsByRunID(t.Context(), run.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 1)

		job := jobs[0]

		// If the job has a task ID, it means it has started and should have logs
		if job.TaskID > 0 {
			// Test getting logs for a job
			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs",
				repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)

			// Verify content type
			assert.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))

			// Verify content disposition header is inline for API
			contentDisposition := resp.Header().Get("Content-Disposition")
			assert.Contains(t, contentDisposition, "inline")
			assert.Contains(t, contentDisposition, ".log")
		} else {
			// Job hasn't started yet, should return 404
			req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs",
				repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		}

		// Test non-existent job logs
		req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/999/logs",
			repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)

		// Test non-existent run logs
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/9999/jobs/0/logs",
			repo.OwnerName, repo.Name).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusNotFound)
	})
}

func TestAPIGetActionJobAuth(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		user4 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)
		token4 := getUserToken(t, user4.LowerName, auth_model.AccessTokenScopeReadRepository)

		// Create a private repo with an action workflow
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-job-auth-test-repo",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/test.yml",
					ContentReader: strings.NewReader(`name: test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`),
				},
			})
		defer f()

		// Make the repository private
		repo.IsPrivate = true
		err := repo_model.UpdateRepositoryCols(db.DefaultContext, repo, "is_private")
		require.NoError(t, err)

		// Wait for the workflow to create a run
		assert.Eventually(t, func() bool {
			count := unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID})
			return count == 1
		}, 1*time.Minute, 1*time.Second, "workflow run should be created within 1 minute")
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})

		// Test with repository owner (should have access)
		req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0",
			repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token)
		MakeRequest(t, req, http.StatusOK)

		// Test with non-owner (should not have access to private repo)
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0",
			repo.OwnerName, repo.Name, run.Index).AddTokenAuth(token4)
		MakeRequest(t, req, http.StatusNotFound)

		// Test without token (should not have access)
		req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/actions/runs/%d/jobs/0",
			repo.OwnerName, repo.Name, run.Index)
		MakeRequest(t, req, http.StatusNotFound)
	})
}
