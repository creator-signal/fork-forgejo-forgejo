// Copyright 2017 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"reflect"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/git"
	"forgejo.org/modules/util"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_localizedExtensions(t *testing.T) {
	tests := []struct {
		name              string
		ext               string
		languageCode      string
		wantLocalizedExts []string
	}{
		{
			name:              "empty language",
			ext:               ".md",
			wantLocalizedExts: []string{".md"},
		},
		{
			name:              "No region - lowercase",
			languageCode:      "en",
			ext:               ".csv",
			wantLocalizedExts: []string{".en.csv", ".csv"},
		},
		{
			name:              "No region - uppercase",
			languageCode:      "FR",
			ext:               ".txt",
			wantLocalizedExts: []string{".fr.txt", ".txt"},
		},
		{
			name:              "With region - lowercase",
			languageCode:      "en-us",
			ext:               ".md",
			wantLocalizedExts: []string{".en-us.md", ".en_us.md", ".en.md", "_en.md", ".md"},
		},
		{
			name:              "With region - uppercase",
			languageCode:      "en-CA",
			ext:               ".MD",
			wantLocalizedExts: []string{".en-ca.MD", ".en_ca.MD", ".en.MD", "_en.MD", ".MD"},
		},
		{
			name:              "With region - all uppercase",
			languageCode:      "ZH-TW",
			ext:               ".md",
			wantLocalizedExts: []string{".zh-tw.md", ".zh_tw.md", ".zh.md", "_zh.md", ".md"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotLocalizedExts := localizedExtensions(tt.ext, tt.languageCode); !reflect.DeepEqual(gotLocalizedExts, tt.wantLocalizedExts) {
				t.Errorf("localizedExtensions() = %v, want %v", gotLocalizedExts, tt.wantLocalizedExts)
			}
		})
	}
}

func Test_WhenViewingRepoWithTagsButNotBranchesPushed_DoesNotRedirect(t *testing.T) {
	unittest.PrepareTestEnv(t)

	ctx, _ := contexttest.MockContext(t, "user2/repo1")
	contexttest.LoadUser(t, ctx, 2)
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	// create a commit
	_, _, err := git.NewCommand(ctx, "commit", "--allow-empty", "--allow-empty-message", "--message", "").RunStdString(&git.RunOpts{})
	require.NoError(t, err)

	// create a tag
	tagName := util.CryptoRandomString(util.RandomStringLow)
	_, _, err = git.NewCommand(ctx, "tag").AddDynamicArguments(tagName).RunStdString(&git.RunOpts{})
	require.NoError(t, err)

	renderHomeCode(ctx)

	// delete the tag
	_, _, err = git.NewCommand(ctx, "tag", "-d").AddDynamicArguments(tagName).RunStdString(&git.RunOpts{})
	require.NoError(t, err)

	assert.Equal(t, 200, ctx.Resp.Status())
}
