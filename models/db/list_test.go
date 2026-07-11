// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package db_test

import (
	"testing"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"xorm.io/builder"
)

type mockListOptions struct {
	db.ListOptions
}

func (opts mockListOptions) IsListAll() bool {
	return true
}

func (opts mockListOptions) ToConds() builder.Cond {
	return builder.NewCond()
}

func TestFind(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	xe, err := unittest.GetXORMEngine()
	require.NoError(t, err)
	require.NoError(t, xe.Sync(&repo_model.RepoUnit{}))

	var repoUnitCount int
	_, err = db.GetEngine(db.DefaultContext).SQL("SELECT COUNT(*) FROM repo_unit").Get(&repoUnitCount)
	require.NoError(t, err)
	assert.NotEmpty(t, repoUnitCount)

	opts := mockListOptions{}
	repoUnits, err := db.Find[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Len(t, repoUnits, repoUnitCount)

	cnt, err := db.Count[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.EqualValues(t, repoUnitCount, cnt)

	repoUnits, newCnt, err := db.FindAndCount[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.Equal(t, cnt, newCnt)
	assert.Len(t, repoUnits, repoUnitCount)
}

// TestFindAndCountPageDefault makes sure that FindAndCount does not silently
// return every row when Page is left unset (its zero value) on a non-ListAll
// query, mirroring the same default that Find already applies.
func TestFindAndCountPageDefault(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())
	xe, err := unittest.GetXORMEngine()
	require.NoError(t, err)
	require.NoError(t, xe.Sync(&repo_model.RepoUnit{}))

	var repoUnitCount int
	_, err = db.GetEngine(db.DefaultContext).SQL("SELECT COUNT(*) FROM repo_unit").Get(&repoUnitCount)
	require.NoError(t, err)
	require.Greater(t, repoUnitCount, 1)

	opts := &db.ListOptions{PageSize: 1} // Page intentionally left as its zero value

	repoUnits, cnt, err := db.FindAndCount[repo_model.RepoUnit](db.DefaultContext, opts)
	require.NoError(t, err)
	assert.EqualValues(t, repoUnitCount, cnt) // the total count must still reflect all rows
	assert.Len(t, repoUnits, 1)               // but the returned page must be limited to PageSize
}
