// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"

	runnerv1 "code.forgejo.org/forgejo/actions-proto/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TestAPIGetActionJobLogs covers the REST endpoint:
//
//	GET /api/v1/repos/{owner}/{repo}/actions/jobs/{job_id}/logs
//
// It verifies happy-path content, cross-repo 404, missing-token 401, the
// read:repository scope gate, Range support, and the step-filter query param.
func TestAPIGetActionJobLogs(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip()
	}
	now := time.Now()
	outcome := &mockTaskOutcome{
		result: runnerv1.Result_RESULT_SUCCESS,
		logRows: []*runnerv1.LogRow{
			{Time: timestamppb.New(now.Add(1 * time.Second)), Content: "first line"},
			{Time: timestamppb.New(now.Add(2 * time.Second)), Content: "second line"},
			{Time: timestamppb.New(now.Add(3 * time.Second)), Content: "third line"},
		},
	}
	workflow := `name: api-job-logs
on:
  push:
    paths:
      - '.gitea/workflows/api-job-logs.yml'
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: echo hello
`
	treePath := ".gitea/workflows/api-job-logs.yml"

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session,
			auth_model.AccessTokenScopeWriteRepository,
			auth_model.AccessTokenScopeWriteUser,
		)

		// Repo A receives the workflow + runs the job.
		apiRepoA := createActionsTestRepo(t, token, "actions-job-logs", false)
		repoA := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: apiRepoA.ID})

		// Repo B is the cross-repo target — used to verify the guard.
		apiRepoB := createActionsTestRepo(t, token, "actions-job-logs-other", false)

		runner := newMockRunner()
		runner.registerAsRepoRunner(t, user2.Name, repoA.Name, "mock-runner", []string{"ubuntu-latest"})

		opts := getWorkflowCreateFileOptions(user2, repoA.DefaultBranch,
			fmt.Sprintf("create %s", treePath), workflow)
		createWorkflowFile(t, token, user2.Name, repoA.Name, treePath, opts)

		task := runner.fetchTask(t)
		runner.execTask(t, task, outcome)

		actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id})
		actionRunJob := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: actionTask.JobID})
		jobID := actionRunJob.ID

		t.Run("happy path: 200 plaintext", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, repoA.Name, jobID,
			)
			req.AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)

			require.Contains(t, resp.Header().Get("Content-Type"), "text/plain")
			assert.Equal(t, "bytes", resp.Header().Get("Accept-Ranges"))

			lines := strings.Split(strings.TrimSpace(resp.Body.String()), "\n")
			require.Len(t, lines, len(outcome.logRows))
			for i, lr := range outcome.logRows {
				assert.Equal(t,
					fmt.Sprintf("%s %s",
						lr.Time.AsTime().Format("2006-01-02T15:04:05.0000000Z07:00"),
						lr.Content),
					lines[i],
				)
			}
		})

		t.Run("cross-repo: 404 when job_id belongs to a different repo", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, apiRepoB.Name, jobID,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("not found: 404 for unknown job_id", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, repoA.Name, jobID+999999,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("no token: 401", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, repoA.Name, jobID,
			)
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		t.Run("wrong scope: 403 without read:repository", func(t *testing.T) {
			// Token with only user scope, no repository access.
			weakToken := getTokenForLoggedInUser(t, session,
				auth_model.AccessTokenScopeReadUser,
			)
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, repoA.Name, jobID,
			)
			req.AddTokenAuth(weakToken)
			MakeRequest(t, req, http.StatusForbidden)
		})

		t.Run("range: 206 partial content", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs",
				user2.Name, repoA.Name, jobID,
			)
			req.AddTokenAuth(token)
			req.Header.Set("Range", "bytes=0-15")
			resp := MakeRequest(t, req, http.StatusPartialContent)
			// [0,15] inclusive = 16 bytes max.
			assert.LessOrEqual(t, resp.Body.Len(), 16)
		})

		t.Run("step filter: 200 returns step's slice", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs?step=0",
				user2.Name, repoA.Name, jobID,
			)
			req.AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)
			// Step 0's slice should be non-empty.
			require.NotEmpty(t, resp.Body.String())
		})

		t.Run("step out of range: 404", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/jobs/%d/logs?step=99",
				user2.Name, repoA.Name, jobID,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		httpContextA := NewAPITestContext(t, user2.Name, repoA.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextA)(t)
		httpContextB := NewAPITestContext(t, user2.Name, apiRepoB.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextB)(t)
	})
}
