// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package actions

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionRun_List(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	runs, total, err := db.FindAndCount[ActionRun](t.Context(), &FindRunOptions{
		ListOptions: db.ListOptions{Page: 1, PageSize: 2},
		RepoID:      int64(4),
	})

	require.NoError(t, err)

	// see models/fixtures/action_run.yml for the test data
	// limit should be respected
	assert.Equal(t, int64(6), total)
	assert.Len(t, runs, 2)

	// the most recent two runs
	assert.Equal(t, int64(896), runs[0].ID)
	assert.Equal(t, int64(895), runs[1].ID)
}
