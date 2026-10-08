// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"testing"

	migration_tests "forgejo.org/models/gitea_migrations/test"
	secret_model "forgejo.org/models/secret"

	"github.com/stretchr/testify/require"
)

func TestActionSecretPairMigration(t *testing.T) {
	x, cleanup := migration_tests.PrepareTestEnv(t, 0, new(secret_model.Secret))
	defer cleanup()
	if x == nil || t.Failed() {
		return
	}
	legacy := &secret_model.Secret{RepoID: 123, Name: secret_model.PairPasswordName, Data: []byte("synthetic-old-ciphertext")}
	_, err := x.Insert(legacy)
	require.NoError(t, err)
	require.NoError(t, addActionSecretPairOperation(x))
	require.NoError(t, addActionSecretPairOperation(x))
	count, err := x.Count(new(secret_model.ActionSecretPairOperation))
	require.NoError(t, err)
	require.Zero(t, count)
	var retained secret_model.Secret
	found, err := x.ID(legacy.ID).Get(&retained)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, *legacy, retained)
	operation := &secret_model.ActionSecretPairOperation{OperationID: "synthetic-migration", RepoID: 123, Purpose: secret_model.PairPurpose, State: secret_model.PairRevoked}
	_, err = x.Insert(operation)
	require.NoError(t, err)
	require.NoError(t, addActionSecretPairOperation(x))
	var tombstone secret_model.ActionSecretPairOperation
	found, err = x.ID(operation.ID).Get(&tombstone)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, *operation, tombstone)
}
