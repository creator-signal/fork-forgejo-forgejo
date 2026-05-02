// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/internal/edu"
	"forgejo.org/models/db"
	"forgejo.org/tests"
	"github.com/stretchr/testify/assert"
)

func TestAdapter_BranchExists(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	a := edu.NewForgejoAdapter()
	ctx := db.DefaultContext

	exists, err := a.BranchExists(ctx, 1, "master")
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = a.BranchExists(ctx, 1, "submits/nonexistent")
	assert.NoError(t, err)
	assert.False(t, exists)
}
