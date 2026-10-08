// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	secret_model "forgejo.org/models/secret"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"
	actions_service "forgejo.org/services/actions"
	repo_service "forgejo.org/services/repository"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationPairRequest() api.ActionSecretPairRequest {
	return api.ActionSecretPairRequest{
		Schema: secret_model.PairRequestSchema, Purpose: secret_model.PairPurpose, OperationID: strings.Repeat("a", 64),
		Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
		Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
	}
}

func TestSecretPairAPIAndUI(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	repository := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: repository.OwnerID})
	session := loginUser(t, owner.Name)
	token := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeWriteRepository)
	input := integrationPairRequest()
	endpoint := "/api/v1/repos/"+repository.FullName()+"/actions/secret-pair-operations/"+input.OperationID
	other := loginUser(t, "user4")
	otherToken := getTokenForLoggedInUser(t, other, auth_model.AccessTokenScopeWriteRepository)
	MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, input).AddTokenAuth(otherToken), http.StatusForbidden)
	readToken := getTokenForLoggedInUser(t, session, auth_model.AccessTokenScopeReadRepository)
	MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, input).AddTokenAuth(readToken), http.StatusForbidden)
	unittest.AssertCount(t, &secret_model.ActionSecretPairOperation{}, 0)
	for _, name := range []string{secret_model.PairUsernameName, secret_model.PairPasswordName} {
		MakeRequest(t, NewRequestWithJSON(t, http.MethodPut, "/api/v1/repos/"+repository.FullName()+"/actions/secrets/"+name, api.CreateOrUpdateSecretOption{Data:"denied"}).AddTokenAuth(token), http.StatusConflict)
	}
	response := MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, input).AddTokenAuth(token), http.StatusCreated)
	var result api.ActionSecretPairResult
	DecodeJSON(t, response, &result)
	assert.False(t, result.Replayed)
	assert.Equal(t, input.Binding, result.Binding)
	assert.NotContains(t, response.Body.String(), input.Secrets.Password)
	assert.NotContains(t, response.Body.String(), input.Secrets.Username)
	response = MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, input).AddTokenAuth(token), http.StatusOK)
	DecodeJSON(t, response, &result)
	assert.True(t, result.Replayed)
	changed := input
	changed.Secrets.Password = "cs-vm-artifact-"+strings.Repeat("f", 43)
	response = MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, changed).AddTokenAuth(token), http.StatusConflict)
	assert.NotContains(t, response.Body.String(), changed.Secrets.Password)
	assert.NotContains(t, response.Body.String(), input.Secrets.Password)
	raw, err := json.Marshal(input)
	require.NoError(t, err)
	duplicate := strings.Replace(string(raw), `"nonce":`, `"nonce":"ignored","nonce":`, 1)
	MakeRequest(t, NewRequestWithBody(t, http.MethodPost, endpoint, strings.NewReader(duplicate)).SetHeader("Content-Type", "application/json").AddTokenAuth(token), http.StatusBadRequest)
	operation := unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{RepoID: repository.ID})
	password := unittest.AssertExistsAndLoadBean(t, &secret_model.Secret{ID: operation.PasswordID})
	for _, name := range []string{secret_model.PairUsernameName, strings.ToLower(secret_model.PairPasswordName)} {
		url := "/api/v1/repos/"+repository.FullName()+"/actions/secrets/"+name
		MakeRequest(t, NewRequestWithJSON(t, http.MethodPut, url, api.CreateOrUpdateSecretOption{Data:"replacement"}).AddTokenAuth(token), http.StatusConflict)
		MakeRequest(t, NewRequest(t, http.MethodDelete, url).AddTokenAuth(token), http.StatusConflict)
	}
	// Actual UI handlers use the same central guards for create, rename and ID deletion.
	settings := "/"+repository.FullName()+"/settings/actions/secrets"
	session.MakeRequest(t, NewRequestWithValues(t, http.MethodPost, settings, map[string]string{"name":secret_model.PairUsernameName,"data":"replacement"}), http.StatusBadRequest)
	session.MakeRequest(t, NewRequestWithValues(t, http.MethodPost, fmt.Sprintf("%s/%d/edit", settings, password.ID), map[string]string{"name":"UNRELATED","data":"replacement"}), http.StatusBadRequest)
	session.MakeRequest(t, NewRequestWithValues(t, http.MethodPost, fmt.Sprintf("%s/%d/delete", settings, password.ID), map[string]string{}), http.StatusBadRequest)
	assert.Equal(t, password, unittest.AssertExistsAndLoadBean(t, &secret_model.Secret{ID: password.ID}))
	// Inactive restore disposition prevents both fresh operation and replay.
	previous := setting.Actions.Enabled
	setting.Actions.Enabled = false
	t.Cleanup(func() { setting.Actions.Enabled = previous })
	MakeRequest(t, NewRequestWithJSON(t, http.MethodPost, endpoint, input).AddTokenAuth(token), http.StatusNotFound)
	setting.Actions.Enabled = previous
	assert.Equal(t, operation, unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{ID: operation.ID}))
}

func TestSecretPairRepositoryDeletionAndSlugReuse(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})
	repository, err := repo_service.CreateRepository(t.Context(), owner, owner, repo_service.CreateRepoOptions{Name:"secret-pair-delete", AutoInit:true})
	require.NoError(t, err)
	require.NoError(t, repo_service.UpdateRepositoryUnits(t.Context(), repository, []repo_model.RepoUnit{{RepoID:repository.ID, Type:unit.TypeActions}}, nil))
	input := integrationPairRequest()
	_, err = secret_model.ApplySecretPairOperation(t.Context(), repository.ID, &input)
	require.NoError(t, err)
	require.NoError(t, repo_service.DeleteRepositoryDirectly(t.Context(), repository.ID, repo_service.DeleteRepositoryOpts{}))
	operation := unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{OperationID:input.OperationID})
	assert.Equal(t, secret_model.PairRevoked, operation.State)
	assert.Equal(t, repository.ID, operation.RepoID)
	unittest.AssertCount(t, &secret_model.Secret{RepoID:repository.ID}, 0)
	recreated, err := repo_service.CreateRepository(t.Context(), owner, owner, repo_service.CreateRepoOptions{Name:repository.Name, AutoInit:true})
	require.NoError(t, err)
	require.NotEqual(t, repository.ID, recreated.ID)
	require.NoError(t, repo_service.UpdateRepositoryUnits(t.Context(), recreated, []repo_model.RepoUnit{{RepoID:recreated.ID, Type:unit.TypeActions}}, nil))
	_, err = secret_model.ApplySecretPairOperation(t.Context(), recreated.ID, &input)
	require.ErrorIs(t, err, secret_model.ErrPairConflict)
	assert.Equal(t, operation, unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{ID:operation.ID}))
}

func TestSecretPairActualRunnerTaskDelivery(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip("existing mock runner SQLite capability")
	}
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		owner := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID:2})
		repository := createFetchTaskTestRepository(t, owner, "pair.yml", "on: [push]\njobs:\n  pair:\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo synthetic\n")
		input := integrationPairRequest()
		_, err := secret_model.ApplySecretPairOperation(t.Context(), repository.ID, &input)
		require.NoError(t, err)
		runner := newMockRunner()
		runner.registerAsRepoRunner(t, owner.Name, repository.Name, "pair-native-mock", []string{"ubuntu-latest"})
		task := runner.fetchTask(t)
		require.NotNil(t, task)
		assert.Equal(t, input.Secrets.Username, task.GetSecrets()[secret_model.PairUsernameName])
		assert.Equal(t, input.Secrets.Password, task.GetSecrets()[secret_model.PairPasswordName])
		storedTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id})
		recovered, err := actions_service.RecoverTasks(t.Context(), []*actions_model.ActionTask{storedTask})
		require.NoError(t, err)
		require.Len(t, recovered, 1)
		assert.Equal(t, input.Secrets.Username, recovered[0].GetSecrets()[secret_model.PairUsernameName])
		assert.Equal(t, input.Secrets.Password, recovered[0].GetSecrets()[secret_model.PairPasswordName])
		beforeFailure := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id})
		previous := setting.Actions.Enabled
		setting.Actions.Enabled = false
		blocked, err := actions_service.RecoverTasks(t.Context(), []*actions_model.ActionTask{storedTask})
		setting.Actions.Enabled = previous
		require.Error(t, err)
		assert.Nil(t, blocked)
		assert.Equal(t, beforeFailure.TokenHash, unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id}).TokenHash)
		operation := unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{RepoID:repository.ID})
		_, err = db.DeleteByID[secret_model.Secret](t.Context(), operation.PasswordID)
		require.NoError(t, err)
		// Recovering the same runner task must re-read pair ownership before exposure.
		storedTask = unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.Id})
		projected, err := actions_service.RecoverTasks(t.Context(), []*actions_model.ActionTask{storedTask})
		require.Error(t, err)
		assert.Nil(t, projected)
	})
}
