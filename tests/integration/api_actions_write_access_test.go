// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

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
	defer unittest.OverrideFixtures("tests/integration/fixtures/TestActionsPullRequestTargetWriteAccess")()
	defer tests.PrepareTestEnv(t)()

	task := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 100})
	require.NoError(t, task.LoadAttributes(db.DefaultContext))

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: task.RepoID})

	token, err := actions.CreateAuthorizationToken(task, nil, false)
	require.NoError(t, err)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/labels", repo.OwnerName, repo.Name)

	req1 := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "test-label-action-pr",
		Color: "#000000",
	})
	req1.AddTokenAuth(token)
	MakeRequest(t, req1, http.StatusForbidden)

	taskTarget := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 200})
	require.NoError(t, taskTarget.LoadAttributes(db.DefaultContext))

	tokenTarget, err := actions.CreateAuthorizationToken(taskTarget, nil, false)
	require.NoError(t, err)

	req2 := NewRequestWithJSON(t, "POST", urlStr, &api.CreateLabelOption{
		Name:  "test-label-action-pr-target",
		Color: "#000000",
	})
	req2.AddTokenAuth(tokenTarget)
	MakeRequest(t, req2, http.StatusCreated)
}
