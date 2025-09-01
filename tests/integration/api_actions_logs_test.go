// Copyright 2025 The Forgejo Authors. All rights reserved.
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
	"forgejo.org/models/db"
	unit_model "forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/json"
	"forgejo.org/modules/storage"
	api "forgejo.org/modules/structs"
	files_service "forgejo.org/services/repository/files"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestLogs creates test log files for action jobs
func createTestLogs(t *testing.T, taskID int64, logFilename string) {
	t.Helper()

	// Generate test log content - multiples of 2 on each line
	// Logs must have timestamps in the format: 2006-01-02T15:04:05.0000000Z07:00
	var logContent strings.Builder
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	for i := 1; i <= 100; i++ {
		// Add a second for each line to simulate time progression
		timestamp := baseTime.Add(time.Duration(i) * time.Second)
		// Format: timestamp + space + content
		// The format must match exactly what FormatLog produces
		logContent.WriteString(fmt.Sprintf("%s Line %d: value=%d\n",
			timestamp.Format("2006-01-02T15:04:05.0000000Z07:00"), i, i*2))
	}

	// Write to storage
	_, err := storage.Actions.Save(logFilename, strings.NewReader(logContent.String()), -1)
	require.NoError(t, err)
}

func TestAPIActionsJobLogs(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)

		// Create a repo with an action workflow
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-job-logs-test-repo",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/test.yml",
					ContentReader: strings.NewReader(`name: test
on: push
jobs:
  test:
    runs-on: docker
    steps:
      - name: Test Step
        run: echo "Hello"`),
				},
			},
		)
		defer f()

		// Wait for the workflow to be created
		assert.Eventually(t, func() bool {
			count := unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID})
			return count == 1
		}, 30*time.Second, 1*time.Second, "workflow run should be created within 30 seconds")
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})

		// Get jobs for this run
		jobs, err := actions_model.GetRunJobsByRunID(db.DefaultContext, run.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 1)
		job := jobs[0]

		// Get the task to find the log filename
		if job.TaskID > 0 {
			task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: job.TaskID})
			// Create test log file for this task
			createTestLogs(t, task.ID, task.LogFilename)
		} else {
			// Job hasn't started yet, create a task for it
			t.Skip("Job hasn't started yet, skipping log tests")
		}

		// Get job details
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		var jobResp api.ActionRunJobResponse
		DecodeJSON(t, resp, &jobResp)
		assert.Equal(t, job.ID, jobResp.ID)

		// Test full log retrieval (default behavior)
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, "text/plain; charset=utf-8", resp.Header().Get("Content-Type"))
		fullLog := resp.Body.String()
		fullLines := strings.Split(strings.TrimSpace(fullLog), "\n")
		assert.Len(t, fullLines, 100, "Should have 100 lines of logs")
		// Verify content - line 32 should have value 64 (32 * 2)
		assert.Contains(t, fullLines[31], "value=64")

		// Test head parameter - should get first 5 lines
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?head=5",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		headLog := resp.Body.String()
		headLines := strings.Split(strings.TrimSpace(headLog), "\n")
		assert.Len(t, headLines, 5, "Should return exactly 5 lines")
		assert.Contains(t, headLines[0], "value=2", "First line should have value=2")
		assert.Contains(t, headLines[4], "value=10", "Fifth line should have value=10")

		// Test tail parameter - should get last 3 lines
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?tail=3",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		tailLog := resp.Body.String()
		tailLines := strings.Split(strings.TrimSpace(tailLog), "\n")
		assert.Len(t, tailLines, 3, "Should return exactly 3 lines")
		assert.Contains(t, tailLines[0], "value=196", "Line 98 should have value=196")
		assert.Contains(t, tailLines[2], "value=200", "Line 100 should have value=200")

		// Test offset with head - should get 3 lines starting from line 3
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?offset=2&head=3",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		offsetLog := resp.Body.String()
		offsetLines := strings.Split(strings.TrimSpace(offsetLog), "\n")
		assert.Len(t, offsetLines, 3, "Should return exactly 3 lines")
		assert.Contains(t, offsetLines[0], "value=6", "Line 3 should have value=6")
		assert.Contains(t, offsetLines[2], "value=10", "Line 5 should have value=10")

		// Test JSON format
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?format=json&head=2",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)
		assert.Equal(t, "application/json;charset=utf-8", resp.Header().Get("Content-Type"))

		var jsonLines []struct {
			LineNumber int64  `json:"line_number"`
			Timestamp  string `json:"timestamp"`
			Content    string `json:"content"`
		}
		DecodeJSON(t, resp, &jsonLines)
		assert.Len(t, jsonLines, 2, "Should return exactly 2 lines")
		assert.Equal(t, int64(1), jsonLines[0].LineNumber)
		assert.Contains(t, jsonLines[0].Content, "value=2")
		assert.Equal(t, int64(2), jsonLines[1].LineNumber)
		assert.Contains(t, jsonLines[1].Content, "value=4")

		// Test error cases
		// Invalid: both head and tail
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?head=5&tail=5",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)

		// Invalid format
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?format=xml",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		MakeRequest(t, req, http.StatusBadRequest)
	})
}

func TestAPIActionsJobLogsPartial(t *testing.T) {
	onGiteaRun(t, func(t *testing.T, u *url.URL) {
		user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
		token := getUserToken(t, user2.LowerName, auth_model.AccessTokenScopeWriteRepository)

		// Create a repo with an action workflow
		repo, _, f := tests.CreateDeclarativeRepo(t, user2, "api-job-logs-partial-test-repo",
			[]unit_model.Type{unit_model.TypeActions}, nil,
			[]*files_service.ChangeRepoFile{
				{
					Operation: "create",
					TreePath:  ".forgejo/workflows/test.yml",
					ContentReader: strings.NewReader(`name: test
on: push
jobs:
  test:
    runs-on: docker
    steps:
      - name: Test Step
        run: echo "Hello"`),
				},
			},
		)
		defer f()

		// Wait for the workflow to be created
		assert.Eventually(t, func() bool {
			count := unittest.GetCount(t, &actions_model.ActionRun{RepoID: repo.ID})
			return count == 1
		}, 30*time.Second, 1*time.Second, "workflow run should be created within 30 seconds")
		run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{RepoID: repo.ID})

		// Get jobs for this run
		jobs, err := actions_model.GetRunJobsByRunID(db.DefaultContext, run.ID)
		require.NoError(t, err)
		require.GreaterOrEqual(t, len(jobs), 1)
		job := jobs[0]

		// Get the task to find the log filename
		if job.TaskID > 0 {
			task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: job.TaskID})
			// Create test log file for this task
			createTestLogs(t, task.ID, task.LogFilename)
		} else {
			// Job hasn't started yet, skip log tests
			t.Skip("Job hasn't started yet, skipping log tests")
		}

		// Test tail with offset (lines ending at offset)
		req := NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?offset=10&tail=5",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp := MakeRequest(t, req, http.StatusOK)
		offsetTailLog := resp.Body.String()
		// This should return lines 6-10 (5 lines ending at line 10)
		lines := strings.Split(strings.TrimSpace(offsetTailLog), "\n")
		assert.Len(t, lines, 5, "Should return exactly 5 lines")
		assert.Contains(t, lines[0], "value=12", "Line 6 should have value=12")
		assert.Contains(t, lines[4], "value=20", "Line 10 should have value=20")

		// Test JSON format with timestamps
		req = NewRequest(t, "GET", fmt.Sprintf("/api/v1/repos/%s/%s/actions/runs/%d/jobs/0/logs?format=json&tail=2",
			repo.OwnerName, repo.Name, run.Index)).
			AddTokenAuth(token)
		resp = MakeRequest(t, req, http.StatusOK)

		var jsonLines []map[string]any
		err = json.Unmarshal(resp.Body.Bytes(), &jsonLines)
		require.NoError(t, err)
		assert.Len(t, jsonLines, 2, "Should return exactly 2 lines")

		for _, line := range jsonLines {
			assert.Contains(t, line, "line_number")
			assert.Contains(t, line, "timestamp")
			assert.Contains(t, line, "content")
		}

		// Verify the last two lines
		if len(jsonLines) == 2 {
			assert.InEpsilon(t, float64(99), jsonLines[0]["line_number"], 0.001)
			assert.Contains(t, jsonLines[0]["content"], "value=198")
			assert.InEpsilon(t, float64(100), jsonLines[1]["line_number"], 0.001)
			assert.Contains(t, jsonLines[1]["content"], "value=200")
		}
	})
}
