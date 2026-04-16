package integration

import (
	"fmt"
	"net/http"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	api "forgejo.org/modules/structs"
	"forgejo.org/services/actions"
	"forgejo.org/tests"

	"github.com/stretchr/testify/require"
)

func TestActionsPullRequestTargetWriteAccess(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	run := &actions_model.ActionRun{
		Title:             "test run",
		RepoID:            repo.ID,
		OwnerID:           repo.OwnerID,
		TriggerUserID:     repo.OwnerID,
		WorkflowID:        "test.yml",
		TriggerEvent:      "pull_request",
		IsForkPullRequest: true,
	}
	require.NoError(t, db.Insert(db.DefaultContext, run))

	job := &actions_model.ActionRunJob{
		RunID:             run.ID,
		RepoID:            repo.ID,
		Name:              "test job",
		Attempt:           1,
		IsForkPullRequest: true,
	}
	require.NoError(t, db.Insert(db.DefaultContext, job))

	task := &actions_model.ActionTask{
		JobID:             job.ID,
		RepoID:            repo.ID,
		Status:            actions_model.StatusRunning,
		IsForkPullRequest: true,
	}
	require.NoError(t, db.Insert(db.DefaultContext, task))

	task.Job = job
	task.Job.Run = run

	token, err := actions.CreateAuthorizationToken(task, nil, false)
	require.NoError(t, err)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/labels", repo.OwnerName, repo.Name)

	req1 := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "test-label-action-pr",
		Color: "#000000",
	})
	req1.Header.Set("Authorization", "Bearer "+token)
	MakeRequest(t, req1, http.StatusForbidden)

	run.TriggerEvent = "pull_request_target"
	_, err = db.GetEngine(db.DefaultContext).ID(run.ID).Cols("trigger_event").Update(run)
	require.NoError(t, err)

	req2 := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "test-label-action-pr-target",
		Color: "#000000",
	})
	req2.Header.Set("Authorization", "Bearer "+token)
	MakeRequest(t, req2, http.StatusCreated)
}
