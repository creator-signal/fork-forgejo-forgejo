// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repository

import (
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForkRepository(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	// user 13 has already forked repo10
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 13})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10})

	fork, err := ForkRepositoryAndUpdates(git.DefaultContext, user, user, ForkRepoOptions{
		BaseRepo:    repo,
		Name:        "test",
		Description: "test",
	})
	assert.Nil(t, fork)
	require.Error(t, err)
	assert.True(t, IsErrForkAlreadyExist(err))

	// user not reached maximum limit of repositories
	assert.False(t, repo_model.IsErrReachLimitOfRepo(err))

	// change AllowForkWithoutMaximumLimit to false for the test
	defer test.MockVariableValue(&setting.Repository.AllowForkWithoutMaximumLimit, false)()
	// user has reached maximum limit of repositories
	user.MaxRepoCreation = 0
	fork2, err := ForkRepositoryAndUpdates(git.DefaultContext, user, user, ForkRepoOptions{
		BaseRepo:    repo,
		Name:        "test",
		Description: "test",
	})
	assert.Nil(t, fork2)
	assert.True(t, repo_model.IsErrReachLimitOfRepo(err))
}

func TestGetOrgUserHasForkedRepo(t *testing.T) {
	defer unittest.OverrideFixtures("models/repo/TestGetOrgUserHasForkedRepo")()
	require.NoError(t, unittest.PrepareTestDatabase())

	// orgUser3 has repo 65 forked from repo63
	repo63, err := repo_model.GetRepositoryByID(db.DefaultContext, 63)
	require.NoError(t, err)
	require.NotNil(t, repo63)

	// check that user 2 who belongs to org 3
	user, _ := user_model.GetUserByID(db.DefaultContext, 2)
	is, err := organization.IsOrganizationMember(db.DefaultContext, 3, user.ID)
	require.NoError(t, err)
	require.True(t, is)

	// check that we can get repo 65 via user 2 who belongs to org 3
	require.True(t, OrgHasForkedRepo(db.DefaultContext, 2, repo63.ID))
}
