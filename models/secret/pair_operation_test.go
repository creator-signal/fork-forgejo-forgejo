// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package secret

import (
	"context"
	"strings"
	"sync"
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	api "forgejo.org/modules/structs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pairTestRequest() *api.ActionSecretPairRequest {
	return &api.ActionSecretPairRequest{
		Schema: PairRequestSchema, Purpose: PairPurpose, OperationID: strings.Repeat("a", 64),
		Binding: api.ActionSecretPairBinding{OwnershipID: strings.Repeat("b", 64), TransactionID: strings.Repeat("c", 64), Nonce: strings.Repeat("d", 64)},
		Secrets: api.ActionSecretPairMaterial{Username: "vm-image-publisher", Password: "cs-vm-artifact-"+strings.Repeat("e", 43)},
	}
}

func preparePairDatabase(t *testing.T) {
	t.Helper()
	require.NoError(t, unittest.PrepareTestDatabase())
	previous := setting.Actions.Enabled
	setting.Actions.Enabled = true
	t.Cleanup(func() { setting.Actions.Enabled = previous })
	exists, err := db.GetEngine(t.Context()).Get(&repo_model.RepoUnit{RepoID: 1, Type: unit.TypeActions})
	require.NoError(t, err)
	if !exists { require.NoError(t, db.Insert(t.Context(), &repo_model.RepoUnit{RepoID: 1, Type: unit.TypeActions})) }
}

func TestSecretPairAtomicReplayAndGuards(t *testing.T) {
	preparePairDatabase(t)
	request := pairTestRequest()
	result, err := ApplySecretPairOperation(t.Context(), 1, request)
	require.NoError(t, err)
	assert.False(t, result.Replayed)
	operation := unittest.AssertExistsAndLoadBean(t, &ActionSecretPairOperation{OperationID: request.OperationID})
	username := unittest.AssertExistsAndLoadBean(t, &Secret{ID: operation.UsernameID})
	password := unittest.AssertExistsAndLoadBean(t, &Secret{ID: operation.PasswordID})
	assert.NotContains(t, string(operation.PasswordData), request.Secrets.Password)
	assert.NotContains(t, string(password.Data), request.Secrets.Password)
	triggers := []string{}
	if setting.Database.Type.IsSQLite3() {
		for _, table := range []string{"secret", "action_secret_pair_operation"} {
			for _, verb := range []string{"INSERT", "UPDATE", "DELETE"} {
				name := "pair_no_write_"+table+"_"+strings.ToLower(verb)
				_, err := db.Exec(t.Context(), "CREATE TRIGGER "+name+" BEFORE "+verb+" ON "+table+" BEGIN SELECT RAISE(ABORT, 'synthetic replay write denied'); END")
				require.NoError(t, err)
				triggers = append(triggers, name)
				t.Cleanup(func() { _, err := db.Exec(context.WithoutCancel(t.Context()), "DROP TRIGGER IF EXISTS "+name); require.NoError(t, err) })
			}
		}
	}
	result, err = ApplySecretPairOperation(t.Context(), 1, request)
	require.NoError(t, err)
	assert.True(t, result.Replayed)
	for _, name := range triggers {
		_, err := db.Exec(t.Context(), "DROP TRIGGER "+name)
		require.NoError(t, err)
	}
	assert.Equal(t, operation, unittest.AssertExistsAndLoadBean(t, &ActionSecretPairOperation{ID: operation.ID}))
	assert.Equal(t, username, unittest.AssertExistsAndLoadBean(t, &Secret{ID: username.ID}))
	assert.Equal(t, password, unittest.AssertExistsAndLoadBean(t, &Secret{ID: password.ID}))
	for _, name := range []string{PairUsernameName, PairPasswordName, strings.ToLower(PairPasswordName)} {
		_, err := InsertEncryptedSecret(t.Context(), 0, 1, name, "replacement")
		require.ErrorIs(t, err, ErrManagedSecret)
		_, err = InsertEncryptedSecret(t.Context(), 2, 0, name, "shadow")
		require.ErrorIs(t, err, ErrManagedSecret)
	}
	ordinary, err := InsertEncryptedSecret(t.Context(), 0, 1, "UNRELATED", "original")
	require.NoError(t, err)
	ordinary.Name = strings.ToLower(PairUsernameName)
	require.ErrorIs(t, UpdateSecret(t.Context(), ordinary), ErrManagedSecret)
	// Caller-supplied Name cannot conceal the actual managed target row.
	concealed := *password
	concealed.Name = "UNRELATED"
	concealed.SetData("replacement")
	require.ErrorIs(t, UpdateSecret(t.Context(), &concealed, "data"), ErrManagedSecret)
	require.ErrorIs(t, DeleteSecret(t.Context(), password.ID), ErrManagedSecret)
	assert.Equal(t, password, unittest.AssertExistsAndLoadBean(t, &Secret{ID: password.ID}))
	projected, err := FetchActionSecrets(t.Context(), 2, 1)
	require.NoError(t, err)
	assert.Equal(t, request.Secrets.Password, projected[PairPasswordName])
	require.NoError(t, ValidateManagedSecretProjection(t.Context(), 2, 1, projected))
	require.ErrorIs(t, ValidateManagedSecretProjection(t.Context(), 2, 1, map[string]string{PairUsernameName: request.Secrets.Username}), ErrPairConflict)
	projected[PairPasswordName] = "substitution"
	require.ErrorIs(t, ValidateManagedSecretProjection(t.Context(), 2, 1, projected), ErrPairConflict)
	changed := *request
	changed.Secrets.Password = "cs-vm-artifact-"+strings.Repeat("f", 43)
	_, err = ApplySecretPairOperation(t.Context(), 1, &changed)
	require.ErrorIs(t, err, ErrPairConflict)
	changed = *request
	changed.Binding.Nonce = strings.Repeat("f", 64)
	_, err = ApplySecretPairOperation(t.Context(), 1, &changed)
	require.ErrorIs(t, err, ErrPairConflict)
	changed = *request
	changed.OperationID = strings.Repeat("f", 64)
	_, err = ApplySecretPairOperation(t.Context(), 1, &changed)
	require.ErrorIs(t, err, ErrPairConflict)
}

func TestSecretPairUnknownRowsAndCorruption(t *testing.T) {
	for _, scope := range []string{"repository", "owner"} {
		t.Run(scope, func(t *testing.T) {
			preparePairDatabase(t)
			row := &Secret{RepoID: 1, Name: PairPasswordName}
			if scope == "owner" {
				row.RepoID, row.OwnerID = 0, 2
			}
			require.NoError(t, db.Insert(t.Context(), row))
			row.SetData("legacy")
			_, err := db.GetEngine(t.Context()).ID(row.ID).Cols("data").Update(row)
			require.NoError(t, err)
			_, err = FetchActionSecrets(t.Context(), 2, 1)
			require.ErrorIs(t, err, ErrPairConflict)
			_, err = ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
			require.ErrorIs(t, err, ErrPairConflict)
		})
	}
	for _, mutation := range []string{"missing", "replaced", "ciphertext", "name"} {
		t.Run(mutation, func(t *testing.T) {
			preparePairDatabase(t)
			request := pairTestRequest()
			_, err := ApplySecretPairOperation(t.Context(), 1, request)
			require.NoError(t, err)
			operation := unittest.AssertExistsAndLoadBean(t, &ActionSecretPairOperation{OperationID: request.OperationID})
			row := unittest.AssertExistsAndLoadBean(t, &Secret{ID: operation.PasswordID})
			switch mutation {
			case "missing", "replaced":
				_, err = db.DeleteByID[Secret](t.Context(), row.ID)
				require.NoError(t, err)
				if mutation == "replaced" {
					row.ID = 0
					require.NoError(t, db.Insert(t.Context(), row))
					row.SetData(request.Secrets.Password)
					_, err = db.GetEngine(t.Context()).ID(row.ID).Cols("data").Update(row)
					require.NoError(t, err)
				}
			case "ciphertext":
				row.Data = []byte("invalid")
				_, err = db.GetEngine(t.Context()).ID(row.ID).Cols("data").Update(row)
				require.NoError(t, err)
			case "name":
				row.Name = strings.ToLower(row.Name)
				_, err = db.GetEngine(t.Context()).ID(row.ID).Cols("name").Update(row)
				require.NoError(t, err)
			}
			_, err = ApplySecretPairOperation(t.Context(), 1, request)
			require.ErrorIs(t, err, ErrPairConflict)
			_, err = FetchActionSecrets(t.Context(), 2, 1)
			require.ErrorIs(t, err, ErrPairConflict)
		})
	}
}

func TestSecretPairAtomicFailureAndConcurrency(t *testing.T) {
	preparePairDatabase(t)
	if !setting.Database.Type.IsSQLite3() {
		t.Skip("SQLite trigger fault injection")
	}
	_, err := db.Exec(t.Context(), "CREATE TRIGGER pair_password_fault BEFORE INSERT ON secret WHEN NEW.name = 'ZOT_VM_ARTIFACT_PASSWORD' BEGIN SELECT RAISE(ABORT, 'synthetic pair fault'); END")
	require.NoError(t, err)
	t.Cleanup(func() { _, err := db.Exec(context.WithoutCancel(t.Context()), "DROP TRIGGER IF EXISTS pair_password_fault"); require.NoError(t, err) })
	_, err = ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
	require.ErrorIs(t, err, ErrPairConflict)
	unittest.AssertCount(t, &ActionSecretPairOperation{}, 0)
	unittest.AssertCount(t, &Secret{RepoID: 1, Name: PairUsernameName}, 0)
	unittest.AssertCount(t, &Secret{RepoID: 1, Name: PairPasswordName}, 0)
	_, err = db.Exec(t.Context(), "DROP TRIGGER pair_password_fault")
	require.NoError(t, err)
	var group sync.WaitGroup
	start := make(chan struct{})
	results := make(chan error, 8)
	for range 8 {
		group.Go(func() {
			<-start
			_, err := ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
			results <- err
		})
	}
	close(start)
	group.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil { success++ } else {
			require.ErrorIs(t, err, ErrPairConflict)
		}
	}
	require.Positive(t, success)
	unittest.AssertCount(t, &ActionSecretPairOperation{RepoID: 1}, 1)
	unittest.AssertCount(t, &Secret{RepoID: 1, Name: PairUsernameName}, 1)
	unittest.AssertCount(t, &Secret{RepoID: 1, Name: PairPasswordName}, 1)
	result, err := ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
	require.NoError(t, err)
	assert.True(t, result.Replayed)
}

func TestSecretPairDisabledAndRevocation(t *testing.T) {
	preparePairDatabase(t)
	setting.Actions.Enabled = false
	_, err := ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
	require.ErrorIs(t, err, ErrPairDisabled)
	unittest.AssertCount(t, &ActionSecretPairOperation{}, 0)
	setting.Actions.Enabled = true
	_, err = ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
	require.NoError(t, err)
	setting.Actions.Enabled = false
	_, err = FetchActionSecrets(t.Context(), 2, 1)
	require.ErrorIs(t, err, ErrPairDisabled)
	setting.Actions.Enabled = true
	// A failed enclosing delete transaction must roll back revocation too.
	require.ErrorIs(t, db.WithTx(t.Context(), func(ctx context.Context) error {
		require.NoError(t, RevokeSecretPairForRepositoryDeletion(ctx, 1))
		return ErrPairConflict
	}), ErrPairConflict)
	require.NoError(t, db.WithTx(t.Context(), func(ctx context.Context) error { return RevokeSecretPairForRepositoryDeletion(ctx, 1) }))
	operation := unittest.AssertExistsAndLoadBean(t, &ActionSecretPairOperation{RepoID: 1})
	assert.Equal(t, PairRevoked, operation.State)
	_, err = ApplySecretPairOperation(t.Context(), 1, pairTestRequest())
	require.ErrorIs(t, err, ErrPairConflict)
	_, err = FetchActionSecrets(t.Context(), 2, 1)
	require.ErrorIs(t, err, ErrPairConflict)
	// Different original numeric repository cannot reuse the global operation ID.
	_, err = ApplySecretPairOperation(t.Context(), 2, pairTestRequest())
	require.Error(t, err)
	assert.Equal(t, operation, unittest.AssertExistsAndLoadBean(t, &ActionSecretPairOperation{ID: operation.ID}))
}
