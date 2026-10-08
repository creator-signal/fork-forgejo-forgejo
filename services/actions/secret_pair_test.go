// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package actions

import (
	"context"
	"io"
	"bytes"
	"crypto/sha256"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	secret_model "forgejo.org/models/secret"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

	"code.forgejo.org/xorm/xorm"
	"code.forgejo.org/xorm/xorm/names"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretPairActualTaskProjection(t *testing.T) {
	defer unittest.OverrideFixtures("services/actions/TestGetSecretsOfJob")()
	require.NoError(t, unittest.PrepareTestDatabase())
	oldEnabled := setting.Actions.Enabled
	setting.Actions.Enabled = true
	t.Cleanup(func() { setting.Actions.Enabled = oldEnabled })
	request := &api.ActionSecretPairRequest{
		Schema: secret_model.PairRequestSchema, Purpose: secret_model.PairPurpose, OperationID: strings.Repeat("a", 64),
		Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
		Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
	}
	_, err := secret_model.ApplySecretPairOperation(t.Context(), 63, request)
	require.NoError(t, err)
	for _, id := range []int64{600, 602, 605} {
		job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: id})
		task := &actions_model.ActionTask{JobID: job.ID, Job: job, Token: "synthetic-task-token"}
		task.GenerateToken()
		// Exercise actual persisted task/job/run graph, not a DTO observation flag.
		require.NoError(t, db.Insert(t.Context(), task))
		loaded := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: task.ID})
		loaded.Job = job
		loaded.Token = task.Token
		secrets, err := getSecretsOfTask(t.Context(), loaded)
		require.NoError(t, err)
		assert.Equal(t, request.Secrets.Password, secrets[secret_model.PairPasswordName])
		assert.Equal(t, task.Token, secrets["FORGEJO_TOKEN"])
	}
	forkJob := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 604})
	forkSecrets, err := getSecretsOfJob(t.Context(), forkJob)
	require.NoError(t, err)
	assert.Empty(t, forkSecrets)
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 600})
	setting.Actions.Enabled = false
	secrets, err := getSecretsOfTask(t.Context(), &actions_model.ActionTask{Job: job, Token: "must-not-project"})
	require.ErrorIs(t, err, secret_model.ErrPairDisabled)
	assert.Nil(t, secrets)
	setting.Actions.Enabled = true
	operation := unittest.AssertExistsAndLoadBean(t, &secret_model.ActionSecretPairOperation{RepoID: 63})
	_, err = db.DeleteByID[secret_model.Secret](t.Context(), operation.PasswordID)
	require.NoError(t, err)
	secrets, err = getSecretsOfTask(t.Context(), &actions_model.ActionTask{Job: job, Token: "must-not-project"})
	require.Error(t, err)
	assert.Nil(t, secrets)
}

func TestSecretPairWorkflowCallAST(t *testing.T) {
	caller := map[string]string{secret_model.PairUsernameName: "vm-image-publisher", secret_model.PairPasswordName: "synthetic-private", "OTHER": "ordinary"}
	for _, value := range []string{
		"literal secrets.ZOT_VM_ARTIFACT_PASSWORD", "${{ secrets.OTHER }}", "${{ secrets['OTHER'] }}",
		"prefix ${{ format('{0}', secrets.OTHER) }} suffix", "${{ 'secrets.ZOT_VM_ARTIFACT_PASSWORD' }}",
	} {
		require.NoError(t, validatePairWorkflowCallMapping(map[string]string{"ALIAS": value}, caller))
	}
	for _, value := range []string{
		"${{ secrets.ZOT_VM_ARTIFACT_PASSWORD }}", "${{ SeCrEtS.zot_vm_artifact_username }}",
		"${{ secrets['ZOT_VM_ARTIFACT_PASSWORD'] }}", "${{ secrets[format('{0}', 'ZOT_VM_ARTIFACT_PASSWORD')] }}",
		"${{ toJSON(secrets) }}", "${{ join(secrets.*, ',') }}", "${{ secrets[inputs.name] }}",
		"${{ false && secrets.ZOT_VM_ARTIFACT_PASSWORD }}", "${{ secrets.OTHER }} ${{ secrets.ZOT_VM_ARTIFACT_PASSWORD }}",
		"${{ secrets[ }}",
	} {
		require.ErrorIs(t, validatePairWorkflowCallMapping(map[string]string{"ALIAS": value}, caller), secret_model.ErrPairConflict)
	}
	require.ErrorIs(t, validatePairWorkflowCallMapping(map[string]string{secret_model.PairPasswordName: "literal"}, caller), secret_model.ErrPairConflict)
	// No managed pair exists: an unrelated dynamic mapping retains its behavior.
	require.NoError(t, validatePairWorkflowCallMapping(map[string]string{"ALIAS": "${{ secrets[inputs.name] }}"}, map[string]string{"OTHER":"ordinary"}))
}

func TestSecretPairWorkflowCallActualMapping(t *testing.T) {
	for _, expression := range []string{"${{ secrets.ZOT_VM_ARTIFACT_PASSWORD }}", "${{ secrets['ZOT_VM_ARTIFACT_PASSWORD'] }}", "${{ toJSON(secrets) }}"} {
		t.Run(expression, func(t *testing.T) {
			defer unittest.OverrideFixtures("services/actions/TestGetSecretsOfJob")()
			require.NoError(t, unittest.PrepareTestDatabase())
			request := &api.ActionSecretPairRequest{
				Schema: secret_model.PairRequestSchema, Purpose: secret_model.PairPurpose, OperationID: strings.Repeat("a", 64),
				Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
				Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
			}
			_, err := secret_model.ApplySecretPairOperation(t.Context(), 63, request)
			require.NoError(t, err)
			outer := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 606})
			outer.WorkflowPayload = []byte(strings.Replace(string(outer.WorkflowPayload), "secrets: inherit", "secrets:\n        ALIAS: \""+expression+"\"", 1))
			_, err = db.GetEngine(t.Context()).ID(outer.ID).Cols("workflow_payload").Update(outer)
			require.NoError(t, err)
			inner := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 605})
			secrets, err := getSecretsOfTask(t.Context(), &actions_model.ActionTask{Job: inner, Token: "must-not-project"})
			require.Error(t, err)
			assert.Nil(t, secrets)
		})
	}
}

// These private native fixtures prove inactive closed-copy-reopen only. They do
// not establish original provider history or permission to re-enable Actions.
func TestSecretPairInactivePhysicalSQLiteRestore(t *testing.T) {
	if !setting.Database.Type.IsSQLite3() {
		t.Skip("physical SQLite native fixture")
	}
	defer unittest.OverrideFixtures("services/actions/TestGetSecretsOfJob")()
	require.NoError(t, unittest.PrepareTestDatabase())
	original, err := db.GetMasterEngine(db.GetEngine(db.DefaultContext))
	require.NoError(t, err)
	originalEnabled := setting.Actions.Enabled
	repository := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 63})
	run := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRun{ID: 900})
	job := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionRunJob{ID: 600})
	var owned *xorm.Engine
	t.Cleanup(func() {
		if owned != nil {
			require.NoError(t, owned.Close())
		}
		db.SetDefaultEngine(context.Background(), original)
		setting.Actions.Enabled = originalEnabled
	})
	open := func(path string) *xorm.Engine {
		x, err := xorm.NewEngine("sqlite3", "file:"+filepath.ToSlash(path)+"?_txlock=immediate")
		require.NoError(t, err)
		x.SetMapper(names.GonicMapper{})
		owned = x
		db.SetDefaultEngine(context.Background(), x)
		return x
	}
	// Read only after Close drained the actual connection; neither dumps nor
	// rewritten SQL stand in for the original physical snapshot.
	readClosed := func(path string) []byte {
		info, err := os.Lstat(path)
		require.NoError(t, err)
		require.True(t, info.Mode().IsRegular() && info.Size() > 0 && info.Size() <= 4<<20)
		file, err := os.Open(path)
		require.NoError(t, err)
		defer file.Close()
		buffer := make([]byte, info.Size()+1)
		n, err := file.ReadAt(buffer, 0)
		require.ErrorIs(t, err, io.EOF)
		require.Equal(t, int(info.Size()), n)
		return buffer[:n]
	}
	for _, state := range []string{"BeforeCreate", "Committed", "Revoked"} {
		t.Run(state, func(t *testing.T) {
			t.Cleanup(func() {
				if owned != nil {
					require.NoError(t, owned.Close())
					owned = nil
				}
				db.SetDefaultEngine(context.Background(), original)
				setting.Actions.Enabled = originalEnabled
				})
			dir := t.TempDir()
			originalPath := filepath.Join(dir, "original.sqlite")
			x := open(originalPath)
			_, err := x.Exec("PRAGMA journal_mode=DELETE")
			require.NoError(t, err)
			require.NoError(t, x.Sync(new(repo_model.Repository), new(repo_model.RepoUnit), new(secret_model.Secret), new(secret_model.ActionSecretPairOperation), new(actions_model.ActionRun), new(actions_model.ActionRunJob), new(actions_model.ActionTask)))
			_, err = x.Insert(repository, &repo_model.RepoUnit{RepoID: 63, Type: unit.TypeActions}, run, job)
			require.NoError(t, err)
			task := &actions_model.ActionTask{JobID: job.ID, Job: job}
			task.GenerateToken()
			_, err = x.Insert(task)
			require.NoError(t, err)
			setting.Actions.Enabled = true
			request := &api.ActionSecretPairRequest{
				Schema: secret_model.PairRequestSchema, Purpose: secret_model.PairPurpose, OperationID: strings.Repeat("a", 64),
				Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
				Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
			}
			if state != "BeforeCreate" {
				_, err = secret_model.ApplySecretPairOperation(t.Context(), 63, request)
				require.NoError(t, err)
			}
			if state == "Revoked" {
				require.NoError(t, db.WithTx(t.Context(), func(ctx context.Context) error {
					if err := secret_model.RevokeSecretPairForRepositoryDeletion(ctx, 63); err != nil {
						return err
					}
					// Match terminal pair-row deletion in the original delete Tx.
					// Repo/run/task remain fixture scaffolding for inactive projection.
					_, err := db.GetEngine(ctx).Where("repo_id = ?", 63).Delete(new(secret_model.Secret))
					return err
				}))
			}
			var beforeRows []secret_model.Secret
			var beforeOperations []secret_model.ActionSecretPairOperation
			require.NoError(t, x.Asc("id").Find(&beforeRows))
			require.NoError(t, x.Asc("id").Find(&beforeOperations))
			if state == "BeforeCreate" {
				require.Empty(t, beforeRows)
				require.Empty(t, beforeOperations)
			} else {
				if state == "Revoked" {
					require.Empty(t, beforeRows)
				} else {
					require.Len(t, beforeRows, 2)
				}
				require.Len(t, beforeOperations, 1)
				expectedState := secret_model.PairActive
				if state == "Revoked" {
					expectedState = secret_model.PairRevoked
				}
				require.True(t, beforeOperations[0].State == expectedState && beforeOperations[0].RepoID == repository.ID)
			}
			require.NoError(t, x.Close())
			owned = nil
			originalBytes := readClosed(originalPath)
			restoredPath := filepath.Join(dir, "restored.sqlite")
			require.NoError(t, os.WriteFile(restoredPath, originalBytes, 0o600))
			restoredBytes := readClosed(restoredPath)
			require.True(t, bytes.Equal(originalBytes, restoredBytes))
			originalHash := sha256.Sum256(originalBytes)
			restoredHash := sha256.Sum256(restoredBytes)
			require.True(t, originalHash == restoredHash)
			setting.Actions.Enabled = false
			x = open(restoredPath)
			var restoredRows []secret_model.Secret
			var restoredOperations []secret_model.ActionSecretPairOperation
			require.NoError(t, x.Asc("id").Find(&restoredRows))
			require.NoError(t, x.Asc("id").Find(&restoredOperations))
			require.True(t, reflect.DeepEqual(beforeRows, restoredRows))
			require.True(t, reflect.DeepEqual(beforeOperations, restoredOperations))
			loadedJob := &actions_model.ActionRunJob{ID: job.ID}
			exists, err := x.Get(loadedJob)
			require.NoError(t, err)
			require.True(t, exists)
			loadedTask := &actions_model.ActionTask{ID: task.ID}
			exists, err = x.Get(loadedTask)
			require.NoError(t, err)
			require.True(t, exists)
			loadedTask.Job = loadedJob
			loadedTask.Token = "must-not-project"
			projected, err := getSecretsOfTask(t.Context(), loadedTask)
			require.ErrorIs(t, err, secret_model.ErrPairDisabled)
			require.Nil(t, projected)
			_, err = secret_model.ApplySecretPairOperation(t.Context(), 63, request)
			require.ErrorIs(t, err, secret_model.ErrPairDisabled)
			projected, err = secret_model.FetchActionSecrets(t.Context(), repository.OwnerID, repository.ID)
			if state == "BeforeCreate" {
				require.NoError(t, err)
				require.Empty(t, projected)
			} else {
				require.ErrorIs(t, err, secret_model.ErrPairDisabled)
				require.Nil(t, projected)
			}
			require.NoError(t, x.Close())
			owned = nil
			require.True(t, bytes.Equal(restoredBytes, readClosed(restoredPath)))
		})
	}
}