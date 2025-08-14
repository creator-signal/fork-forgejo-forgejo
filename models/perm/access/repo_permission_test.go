package access_test

import (
	"testing"

	actions_model "forgejo.org/models/actions"
	"forgejo.org/models/db"
	perm_model "forgejo.org/models/perm"
	"forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unit"
	"forgejo.org/models/unittest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCodeAccess(t *testing.T, mode perm_model.AccessMode, perm *access.Permission) {
	assert.Equal(t, mode, perm.AccessMode)

	if mode > perm_model.AccessModeNone {
		assert.Len(t, perm.Units, 1)
		assert.Equal(t, unit.TypeCode, perm.Units[0].Type)
		assert.Equal(t, mode, perm.UnitsMode[unit.TypeCode])
	} else {
		assert.Len(t, perm.Units, 0)
	}
}

func TestActionTaskCanAccessOwnRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: actionTask.RepoID})

	perm, err := access.GetActionRepoPermission(db.DefaultContext, repo, actionTask)
	require.NoError(t, err)
	assertCodeAccess(t, perm_model.AccessModeWrite, &perm)
}

func TestActionTaskCanAccessPublicRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1})

	perm, err := access.GetActionRepoPermission(db.DefaultContext, repo, actionTask)
	require.NoError(t, err)
	assertCodeAccess(t, perm_model.AccessModeRead, &perm)
}

func TestActionTaskCanAccessPublicRepoOfLimitedOrg(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 38})

	perm, err := access.GetActionRepoPermission(db.DefaultContext, repo, actionTask)
	require.NoError(t, err)
	assertCodeAccess(t, perm_model.AccessModeRead, &perm)
}

func TestActionTaskNoAccessPublicRepoOfPrivateOrg(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 40})

	perm, err := access.GetActionRepoPermission(db.DefaultContext, repo, actionTask)
	require.NoError(t, err)
	assertCodeAccess(t, perm_model.AccessModeNone, &perm)
}

func TestActionTaskNoAccessPrivateRepo(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	actionTask := unittest.AssertExistsAndLoadBean(t, &actions_model.ActionTask{ID: 47})
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2})

	perm, err := access.GetActionRepoPermission(db.DefaultContext, repo, actionTask)
	require.NoError(t, err)
	assertCodeAccess(t, perm_model.AccessModeNone, &perm)
}
