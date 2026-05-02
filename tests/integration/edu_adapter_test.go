// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"testing"

	"forgejo.org/internal/edu"
	git_model "forgejo.org/models/git"
	"forgejo.org/models/db"
	"forgejo.org/models/perm"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
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

func TestAdapter_AddCollaborator(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	a := edu.NewForgejoAdapter()
	ctx := db.DefaultContext
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 4})

	err := a.AddCollaborator(ctx, repo.ID, user.ID, perm.AccessModeWrite)
	assert.NoError(t, err)

	mode, err := access_model.AccessLevel(ctx, user, repo)
	assert.NoError(t, err)
	assert.Equal(t, perm.AccessModeWrite, mode)
}

func TestAdapter_ProtectMainBranch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	a := edu.NewForgejoAdapter()
	ctx := db.DefaultContext
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	err := a.ProtectMainBranch(ctx, repo.ID, "master")
	assert.NoError(t, err)

	pb, err := git_model.GetProtectedBranchRuleByName(ctx, repo.ID, "master")
	assert.NoError(t, err)
	assert.NotNil(t, pb)
	assert.False(t, pb.CanPush)
}
