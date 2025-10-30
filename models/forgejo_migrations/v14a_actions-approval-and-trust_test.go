// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	migration_tests "forgejo.org/models/gitea_migrations/test"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_v14ActionsApprovalAndTrustPopulateTableActionUser(t *testing.T) {
	x, deferable := migration_tests.PrepareTestEnv(t, 0, new(user_model.User), new(repo_model.Repository), new(actions_model.ActionUser), new(actions_model.ActionRun))
	defer deferable()
	if x == nil || t.Failed() {
		return
	}

	require.NoError(t, v14ActionsApprovalAndTrustPopulateTableActionUser(x))

	var users []*actions_model.ActionUser
	require.NoError(t, db.GetEngine(t.Context()).Select("`repo_id`, `user_id`").OrderBy("`id`").Find(&users))
	// See models/gitea_migrations/fixtures/Test_v14ActionsApprovalAndTrustPopulateTableActionUser/action_run.yml
	assert.Equal(t, []*actions_model.ActionUser{
		{
			UserID: 3,
			RepoID: 15,
		},
		{
			UserID: 3,
			RepoID: 63,
		},
		{
			UserID: 4,
			RepoID: 63,
		},
	}, users)
}
