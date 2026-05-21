// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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

// TestAPIGetActionRunLogs covers the REST endpoint:
//
//	GET /api/v1/repos/{owner}/{repo}/actions/runs/{run_id}/logs
//
// It verifies the happy path returns a valid ZIP with per-job entries,
// cross-repo 404, unknown-run 404, missing-token 401, and the
// read:repository scope gate.
func TestAPIGetActionRunLogs(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip()
	}
	now := time.Now()
	outcomeJob1 := &mockTaskOutcome{
		result: runnerv1.Result_RESULT_SUCCESS,
		logRows: []*runnerv1.LogRow{
			{Time: timestamppb.New(now.Add(1 * time.Second)), Content: "job1-output line one"},
			{Time: timestamppb.New(now.Add(2 * time.Second)), Content: "job1-output line two"},
		},
	}
	outcomeJob2 := &mockTaskOutcome{
		result: runnerv1.Result_RESULT_SUCCESS,
		logRows: []*runnerv1.LogRow{
			{Time: timestamppb.New(now.Add(3 * time.Second)), Content: "job2-output line one"},
			{Time: timestamppb.New(now.Add(4 * time.Second)), Content: "job2-output line two"},
		},
	}
	workflow := `name: api-run-logs
on:
  push:
    paths:
      - '.gitea/workflows/api-run-logs.yml'
jobs:
  job1:
    runs-on: ubuntu-latest
    steps:
      - run: echo job1 first line
  job2:
    runs-on: ubuntu-latest
    steps:
      - run: echo job2 first line
`
	treePath := ".gitea/workflows/api-run-logs.yml"

	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		session := loginUser(t, user2.Name)
		token := getTokenForLoggedInUser(t, session,
			auth_model.AccessTokenScopeWriteRepository,
			auth_model.AccessTokenScopeWriteUser,
		)

		// Repo A receives the workflow + runs the jobs.
		apiRepoA := createActionsTestRepo(t, token, "actions-run-logs", false)
		repoA := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: apiRepoA.ID})

		// Repo B is the cross-repo target — used to verify the guard.
		apiRepoB := createActionsTestRepo(t, token, "actions-run-logs-other", false)

		runner := newMockRunner()
		runner.registerAsRepoRunner(t, user2.Name, repoA.Name, "mock-runner", []string{"ubuntu-latest"})

		opts := getWorkflowCreateFileOptions(user2, repoA.DefaultBranch,
			fmt.Sprintf("create %s", treePath), workflow)
		createWorkflowFile(t, token, user2.Name, repoA.Name, treePath, opts)

		// Fetch + execute both jobs. They both belong to the same run.
		task1 := runner.fetchTask(t)
		outcomeForTask1 := outcomeJob1
		outcomeForTask2 := outcomeJob2
		// Detect which job was fetched first by inspecting the ActionRunJob row.
		actionTask1 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task1.Id})
		actionRunJob1 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: actionTask1.JobID})
		if actionRunJob1.JobID == "job2" {
			outcomeForTask1, outcomeForTask2 = outcomeJob2, outcomeJob1
		}
		runner.execTask(t, task1, outcomeForTask1)

		task2 := runner.fetchTask(t)
		runner.execTask(t, task2, outcomeForTask2)

		actionTask2 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task2.Id})
		actionRunJob2 := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: actionTask2.JobID})

		require.Equal(t, actionRunJob1.RunID, actionRunJob2.RunID,
			"both jobs should belong to the same run")
		runID := actionRunJob1.RunID

		t.Run("happy path: 200 valid zip with per-job entries", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/runs/%d/logs",
				user2.Name, repoA.Name, runID,
			)
			req.AddTokenAuth(token)
			resp := MakeRequest(t, req, http.StatusOK)

			assert.Equal(t, "application/zip", resp.Header().Get("Content-Type"))
			assert.Contains(t, resp.Header().Get("Content-Disposition"),
				fmt.Sprintf("run-%d-logs.zip", runID))

			r, err := zip.NewReader(bytes.NewReader(resp.Body.Bytes()), int64(resp.Body.Len()))
			require.NoError(t, err)
			require.Len(t, r.File, 2, "zip should contain one entry per job")

			foundJob1 := false
			foundJob2 := false
			for _, f := range r.File {
				assert.True(t, strings.HasSuffix(f.Name, ".log"),
					"zip entry %q should have a .log extension", f.Name)
				fr, err := f.Open()
				require.NoError(t, err)
				data, err := io.ReadAll(fr)
				require.NoError(t, err)
				require.NoError(t, fr.Close())
				content := string(data)
				if strings.Contains(content, "job1-output") {
					foundJob1 = true
				}
				if strings.Contains(content, "job2-output") {
					foundJob2 = true
				}
			}
			assert.True(t, foundJob1, "zip should contain job1's log content")
			assert.True(t, foundJob2, "zip should contain job2's log content")
		})

		t.Run("cross-repo: 404 when run_id belongs to a different repo", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/runs/%d/logs",
				user2.Name, apiRepoB.Name, runID,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("not found: 404 for unknown run_id", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/runs/%d/logs",
				user2.Name, repoA.Name, runID+999999,
			)
			req.AddTokenAuth(token)
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("no token: 401", func(t *testing.T) {
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/runs/%d/logs",
				user2.Name, repoA.Name, runID,
			)
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		t.Run("wrong scope: 403 without read:repository", func(t *testing.T) {
			// Token with only user scope, no repository access.
			weakToken := getTokenForLoggedInUser(t, session,
				auth_model.AccessTokenScopeReadUser,
			)
			req := NewRequestf(t, "GET",
				"/api/v1/repos/%s/%s/actions/runs/%d/logs",
				user2.Name, repoA.Name, runID,
			)
			req.AddTokenAuth(weakToken)
			MakeRequest(t, req, http.StatusForbidden)
		})

		httpContextA := NewAPITestContext(t, user2.Name, repoA.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextA)(t)
		httpContextB := NewAPITestContext(t, user2.Name, apiRepoB.Name, auth_model.AccessTokenScopeWriteUser)
		doAPIDeleteRepository(httpContextB)(t)
	})
}
