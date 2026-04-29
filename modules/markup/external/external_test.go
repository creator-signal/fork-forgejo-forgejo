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

// TestRender_UserSuppliedStringsNotInArgv pins the contract that
// user-influenced URL fragments (SrcLink / RawLink) reach the external
// renderer ONLY via environment variables, never via argv.
//
// Pre-fix the renderer Command template had $GITEA_PREFIX_SRC /
// $GITEA_PREFIX_RAW substituted into the command string before strings.Fields
// split it. A whitespace-bearing path component (e.g. a filename containing a
// space) split into extra argv tokens, letting an attacker inject arguments
// into asciidoctor / pandoc / etc.
//
// Post-fix, no string substitution happens; the literal "$GITEA_PREFIX_SRC"
// stays in argv and the URL is only available via the environment.
func TestRender_UserSuppliedStringsNotInArgv(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("relies on the POSIX `true` no-op binary")
	}

	var capturedName string
	var capturedArgs []string
	var capturedCmd *exec.Cmd

	defer test.MockVariableValue(&commandContext, func(ctx context.Context, name string, args ...string) *exec.Cmd {
		capturedName = name
		capturedArgs = append([]string(nil), args...)
		// Return a benign command so Render's cmd.Run() succeeds. Render mutates
		// cmd.Env after we return; we inspect it after Render returns.
		capturedCmd = exec.CommandContext(ctx, "true")
		return capturedCmd
	})()

	r := &Renderer{
		MarkupRenderer: &setting.MarkupRenderer{
			MarkupName: "noinject-test",
			Command:    "fake-renderer $GITEA_PREFIX_SRC $GITEA_PREFIX_RAW",
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
	require.NotEmpty(t, srcLink)
	require.NotEmpty(t, rawLink)

	require.NoError(t, r.Render(ctx, strings.NewReader(""), &bytes.Buffer{}))

	assert.Equal(t, "fake-renderer", capturedName)
	assert.Equal(t, []string{"$GITEA_PREFIX_SRC", "$GITEA_PREFIX_RAW"}, capturedArgs)
	assert.NotContains(t, capturedArgs, srcLink, "SrcLink leaked into argv")
	assert.NotContains(t, capturedArgs, rawLink, "RawLink leaked into argv")

	require.NotNil(t, capturedCmd)
	assert.Contains(t, capturedCmd.Env, "GITEA_PREFIX_SRC="+srcLink)
	assert.Contains(t, capturedCmd.Env, "GITEA_PREFIX_RAW="+rawLink)
}
