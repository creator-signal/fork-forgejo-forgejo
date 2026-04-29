// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package external

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"forgejo.org/modules/markup"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubstitutePrefixPlaceholders(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX $VAR placeholder syntax")
	}

	cases := []struct {
		name             string
		args             []string
		srcLink, rawLink string
		want             []string
	}{
		{
			name:    "clean URLs",
			args:    []string{"$GITEA_PREFIX_SRC", "--raw=$GITEA_PREFIX_RAW", "--safe"},
			srcLink: "https://example.com/o/r/src/branch/main/dir/file",
			rawLink: "https://example.com/o/r/raw/branch/main/dir/file",
			want: []string{
				"https://example.com/o/r/src/branch/main/dir/file",
				"--raw=https://example.com/o/r/raw/branch/main/dir/file",
				"--safe",
			},
		},
		{
			name:    "whitespace in srcLink does not split argv",
			args:    []string{"$GITEA_PREFIX_SRC", "--raw=$GITEA_PREFIX_RAW", "--safe"},
			srcLink: "https://example.com/dir/has space/file",
			rawLink: "https://example.com/dir/has space/file.raw",
			want: []string{
				"https://example.com/dir/has space/file",
				"--raw=https://example.com/dir/has space/file.raw",
				"--safe",
			},
		},
		{
			name:    "flag-shaped content in srcLink stays in one token",
			args:    []string{"$GITEA_PREFIX_SRC", "--safe"},
			srcLink: "innocent --evil-flag=evil --another",
			rawLink: "x",
			want: []string{
				"innocent --evil-flag=evil --another",
				"--safe",
			},
		},
		{
			name:    "no placeholder",
			args:    []string{"--no-placeholders", "literal", "args"},
			srcLink: "https://example.com/x",
			rawLink: "https://example.com/y",
			want:    []string{"--no-placeholders", "literal", "args"},
		},
		{
			name:    "embedded placeholder",
			args:    []string{"--prefix=$GITEA_PREFIX_SRC/foo", "--bar"},
			srcLink: "https://example.com",
			rawLink: "z",
			want:    []string{"--prefix=https://example.com/foo", "--bar"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string(nil), tc.args...)
			substitutePrefixPlaceholders(args, tc.srcLink, tc.rawLink)
			assert.Equal(t, tc.want, args)
			assert.Len(t, args, len(tc.args))
		})
	}
}

func TestRender_PrefixPlaceholderSubstitution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses POSIX $VAR placeholder syntax")
	}

	var capturedName string
	var capturedArgs []string
	var capturedCmd *exec.Cmd

	defer test.MockVariableValue(&commandContext, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = append([]string(nil), args...)
		capturedCmd = exec.CommandContext(ctx, "true")
		return capturedCmd
	})()

	const cmdTemplate = "fake-renderer $GITEA_PREFIX_SRC --raw=$GITEA_PREFIX_RAW --safe"

	r := &Renderer{
		MarkupRenderer: &setting.MarkupRenderer{
			MarkupName: "placeholder-test",
			Command:    cmdTemplate,
		},
	}

	ctx := &markup.RenderContext{
		Ctx: context.Background(),
		Links: markup.Links{
			Base:       "https://example.test/owner/repo",
			BranchPath: "branch/main",
			TreePath:   "dir/file",
		},
	}
	srcLink := ctx.Links.SrcLink()
	rawLink := ctx.Links.RawLink()

	require.NoError(t, r.Render(ctx, strings.NewReader(""), &bytes.Buffer{}))

	assert.Equal(t, "fake-renderer", capturedName)
	assert.Equal(t, []string{srcLink, "--raw=" + rawLink, "--safe"}, capturedArgs)

	require.NotNil(t, capturedCmd)
	assert.Contains(t, capturedCmd.Env, "GITEA_PREFIX_SRC="+srcLink)
	assert.Contains(t, capturedCmd.Env, "GITEA_PREFIX_RAW="+rawLink)
}
