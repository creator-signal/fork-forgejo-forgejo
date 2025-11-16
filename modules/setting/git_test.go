// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"os"
	"path/filepath"
	"testing"

	"forgejo.org/modules/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitConfig(t *testing.T) {
	defer test.MockProtect(&Git)()
	defer test.MockProtect(&GitConfig)()

	cfg, err := NewConfigProviderFromData(`
[git.config]
a.b = 1
`)
	require.NoError(t, err)
	loadGitFrom(cfg)
	assert.Equal(t, "1", GitConfig.Options["a.b"])
	assert.Equal(t, "histogram", GitConfig.Options["diff.algorithm"])

	cfg, err = NewConfigProviderFromData(`
[git.config]
diff.algorithm = other
`)
	require.NoError(t, err)
	loadGitFrom(cfg)
	assert.Equal(t, "other", GitConfig.Options["diff.algorithm"])
}

func TestGitReflog(t *testing.T) {
	defer test.MockProtect(&Git)()
	defer test.MockProtect(&GitConfig)()

	// default reflog config without legacy options
	cfg, err := NewConfigProviderFromData(``)
	require.NoError(t, err)
	loadGitFrom(cfg)

	assert.Equal(t, "true", GitConfig.GetOption("core.logAllRefUpdates"))
	assert.Equal(t, "90", GitConfig.GetOption("gc.reflogExpire"))

	// custom reflog config by legacy options
	cfg, err = NewConfigProviderFromData(`
[git.reflog]
ENABLED = false
EXPIRATION = 123
`)
	require.NoError(t, err)
	loadGitFrom(cfg)

	assert.Equal(t, "false", GitConfig.GetOption("core.logAllRefUpdates"))
	assert.Equal(t, "123", GitConfig.GetOption("gc.reflogExpire"))
}

func TestGitSelfTestGitCredentialHelperPath(t *testing.T) {
	defer test.MockProtect(&Git)()
	defer test.MockProtect(&GitConfig)()

	t.Run("Path is file as directory", func(t *testing.T) {
		require.ErrorContains(t, selfTestGitCredentialHelperPath("/dev/null/aaa"), `could not determine if "/dev/null/aaa" is a directory: stat /dev/null/aaa: not a directory`)
	})

	t.Run("Path is under a read-only directory", func(t *testing.T) {
		readOnlyDir := filepath.Join(t.TempDir(), "read-only")
		require.NoError(t, os.Mkdir(readOnlyDir, 0o500))

		require.ErrorContains(t, selfTestGitCredentialHelperPath(filepath.Join(readOnlyDir, "aaa")), `could not create a directory at "`)
	})

	t.Run("Path is a file", func(t *testing.T) {
		require.ErrorContains(t, selfTestGitCredentialHelperPath("/dev/null"), `the path "/dev/null" is not a directory`)
	})

	t.Run("Path is a read-only directory", func(t *testing.T) {
		readOnlyDir := filepath.Join(t.TempDir(), "read-only")
		require.NoError(t, os.Mkdir(readOnlyDir, 0o400))

		require.ErrorContains(t, selfTestGitCredentialHelperPath(readOnlyDir), `could not write self-test program to "`)
	})

	t.Run("Normal", func(t *testing.T) {
		readOnlyDir := filepath.Join(t.TempDir(), "normal")
		require.NoError(t, os.Mkdir(readOnlyDir, 0o700))

		require.NoError(t, selfTestGitCredentialHelperPath(readOnlyDir))
	})
}
