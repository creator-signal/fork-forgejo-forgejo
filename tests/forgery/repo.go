// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgery

import (
	"cmp"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/git"
	repo_service "forgejo.org/services/repository"
	files_service "forgejo.org/services/repository/files"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type CreateRepositoryOptions struct {
	InitFiles    fs.FS            // Content of the initial commit (if nil a simple README.md will be committed).
	ObjectFormat git.ObjectFormat // If nil, SHA1

	// Name          optional.Option[string]
	// EnabledUnits  optional.Option[[]unit_model.Type]
	// DisabledUnits optional.Option[[]unit_model.Type]
	// UnitConfig    optional.Option[map[unit_model.Type]convert.Conversion]
	// WikiBranch    optional.Option[string]
	// AutoInit      optional.Option[bool]
	// IsTemplate    optional.Option[bool]
	// IsPrivate     optional.Option[bool]
}

// CreateRepository returns the repo, the last commit SHA, and a defer-able function to delete the repo.
// owner and opts can be nil
func CreateRepository(t testing.TB, owner *user_model.User, opts *CreateRepositoryOptions) (*repo_model.Repository, string, func()) {
	t.Helper()

	if owner == nil {
		owner = CreateUser(t, nil)
	}
	if opts == nil {
		opts = &CreateRepositoryOptions{}
	}

	repoName := newEntityName("repo-", t.Name())

	gitFormat := cmp.Or(opts.ObjectFormat, git.Sha1ObjectFormat)

	// Create the repository
	repo, err := repo_service.CreateRepositoryDirectly(t.Context(), owner, owner, repo_service.CreateRepoOptions{
		Name:          repoName,
		Description:   "Test Repo",
		AutoInit:      false,
		Gitignores:    "",
		License:       "CC0",
		Readme:        "Default",
		DefaultBranch: "main",
		// IsTemplate:       opts.IsTemplate.Value(),
		ObjectFormatName: gitFormat.Name(),
		// IsPrivate:        opts.IsPrivate.Value(),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, repo)

	var sha string
	{
		// create initial commit
		fsys := cmp.Or(opts.InitFiles, fs.FS(fstest.MapFS{
			"README.md": &fstest.MapFile{
				Data: []byte("# Hello Test"),
			},
		}))
		sha, err = initRepo(owner, repo, gitFormat, fsys, "init")
		require.NoError(t, err)

		// reload the repo since pushing a commit might update the model via the push_update queue (IsEmpty for instance)
		repo, err = repo_model.GetRepositoryByID(t.Context(), repo.ID)
		require.NoError(t, err)
	}

	return repo, sha, func() {
		_ = repo_service.DeleteRepository(t.Context(), owner, repo, false)
	}
}

func initRepo(doer *user_model.User, repo *repo_model.Repository, format git.ObjectFormat, fsys fs.FS, commitMessage string) (string, error) {
	t, err := files_service.NewTemporaryUploadRepository(git.DefaultContext, repo)
	if err != nil {
		return "", err
	}
	defer t.Close()
	if err := t.Init(format.Name()); err != nil {
		return "", err
	}

	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		// Add the object to the database
		f, err := fsys.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		objectHash, err := t.HashObject(f)
		if err != nil {
			return err
		}
		// Add the object to the index
		return t.AddObjectToIndex("100644", objectHash, path)
	}); err != nil {
		return "", err
	}

	treeHash, err := t.WriteTree()
	if err != nil {
		return "", err
	}

	now := time.Now()
	commitHash, err := t.CommitTreeWithDate("", doer, doer, treeHash, commitMessage, false, now, now)
	if err != nil {
		return "", err
	}

	return commitHash, t.Push(doer, commitHash, repo.DefaultBranch)
}
