// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"testing"
	"testing/fstest"

	user_model "forgejo.org/models/user"
	"forgejo.org/routers/web/repo"
	"forgejo.org/services/context"
	"forgejo.org/services/contexttest"
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/assert"
)

func createRepoAndGetContext(t *testing.T, user *user_model.User, files ...string) (*context.Context, func()) {
	t.Helper()

	fsys := make(fstest.MapFS, len(files))
	for _, name := range files {
		fsys[name] = &fstest.MapFile{
			Data: []byte("test"),
		}
	}
	repo, _, f := forgery.CreateRepository(t, user, &forgery.CreateRepositoryOptions{
		InitFiles: fsys,
	})

	ctx, _ := contexttest.MockContext(t, repo.FullName())
	ctx.SetParams(":id", fmt.Sprint(repo.ID))
	contexttest.LoadRepo(t, ctx, repo.ID)
	contexttest.LoadGitRepo(t, ctx)
	contexttest.LoadRepoCommit(t, ctx)

	return ctx, func() {
		f()
		ctx.Repo.GitRepo.Close()
	}
}

func TestRepoView_FindReadme(t *testing.T) {
	t.Parallel()
	_ = forgery.SharedInstance(t)
	user := forgery.CreateUser(t, nil)

	t.Run("PrioOneLocalizedMdReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.en.md", "README.en.org", "README.org", "README.txt", "README.tex", "README.md")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.en.md", file.Name())
	})
	t.Run("PrioTwoMdReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.en.org", "README.org", "README.txt", "README.tex", "README.md")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.md", file.Name())
	})
	t.Run("PrioThreeLocalizedOrgReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.en.org", "README.org", "README.txt", "README.tex")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.en.org", file.Name())
	})
	t.Run("PrioFourOrgReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.org", "README.txt", "README.tex")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.org", file.Name())
	})
	t.Run("PrioFiveTxtReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.txt", "README", "README.tex")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.txt", file.Name())
	})
	t.Run("PrioSixWithoutExtensionReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README", "README.tex")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README", file.Name())
	})
	t.Run("PrioSevenAnyReadme", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user, "README.tex")
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Equal(t, "README.tex", file.Name())
	})
	t.Run("DoNotPickReadmeIfNonPresent", func(t *testing.T) {
		ctx, f := createRepoAndGetContext(t, user)
		defer f()

		tree, _ := ctx.Repo.Commit.SubTree(ctx.Repo.TreePath)
		entries, _ := tree.ListEntries()
		_, file, _ := repo.FindReadmeFileInEntries(ctx, entries, false)

		assert.Nil(t, file)
	})
}

func TestRepoViewFileLines(t *testing.T) {
	t.Parallel()
	sess := forgery.SharedInstance(t).Session()
	repo, _, f := forgery.CreateRepository(t, nil, &forgery.CreateRepositoryOptions{
		InitFiles: fstest.MapFS{
			"test-1": &fstest.MapFile{
				Data: []byte("No newline"),
			},
			"test-2": &fstest.MapFile{
				Data: []byte("No newline\n"),
			},
			"test-3": &fstest.MapFile{
				Data: []byte("Two\nlines"),
			},
			"test-4": &fstest.MapFile{
				Data: []byte("Really two\nlines\n"),
			},
			"empty": &fstest.MapFile{
				Data: []byte(""),
			},
			"seemingly-empty": &fstest.MapFile{
				Data: []byte("\n"),
			},
			"CITATION.cff": &fstest.MapFile{
				Data: []byte(""),
			},
		},
	})
	defer f()
	_ = f

	testEOL := func(t *testing.T, filename string, hasEOL bool) {
		t.Helper()
		htmlDoc := sess.Get(t, repo.Link(), "src/branch/main", filename).HTMLDoc(t)

		fileInfo := htmlDoc.Find(".file-info").Text()
		if hasEOL {
			assert.NotContains(t, fileInfo, "No EOL")
		} else {
			assert.Contains(t, fileInfo, "No EOL")
		}
	}

	t.Run("No EOL", func(t *testing.T) {
		testEOL(t, "test-1", false)
		testEOL(t, "test-3", false)
	})

	t.Run("With EOL", func(t *testing.T) {
		testEOL(t, "test-2", true)
		testEOL(t, "test-4", true)
		testEOL(t, "empty", true)
		testEOL(t, "seemingly-empty", true)
	})
	t.Run("list", func(t *testing.T) {
		htmlDoc := sess.Get(t, repo.Link()).HTMLDoc(t)

		nodes := htmlDoc.Find("#repo-files-table tr")
		t.Run("CITATION.cff", func(t *testing.T) {
			c, ok := nodes.Find(`.name a[title="CITATION.cff"] svg`).Attr("class")
			assert.True(t, ok, "could not find CITATION.cff line")
			assert.Contains(t, c, "octicon-cross-reference")
		})
	})
}
