// Copyright 2026 The Forgejo Authors. All rights reserved.
// Copyright 2019 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package allowlist

import (
	"path/filepath"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/require"
)

func TestMigrateWhiteBlocklist(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	adminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user1"})
	nonAdminUser := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})

	setting.Migrations.AllowedDomains = "github.com"
	setting.Migrations.AllowLocalNetworks = false
	require.NoError(t, Init())

	_, err := IsMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git", nonAdminUser)
	require.Error(t, err)

	_, err = IsMigrateURLAllowed("https://github.com/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	_, err = IsMigrateURLAllowed("https://gITHUb.com/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	setting.Migrations.AllowedDomains = ""
	setting.Migrations.BlockedDomains = "github.com"
	require.NoError(t, Init())

	_, err = IsMigrateURLAllowed("https://gitlab.com/gitlab/gitlab.git", nonAdminUser)
	require.NoError(t, err)

	_, err = IsMigrateURLAllowed("https://github.com/go-gitea/gitea.git", nonAdminUser)
	require.Error(t, err)

	_, err = IsMigrateURLAllowed("https://10.0.0.1/go-gitea/gitea.git", nonAdminUser)
	require.Error(t, err)

	setting.Migrations.AllowLocalNetworks = true
	require.NoError(t, Init())
	_, err = IsMigrateURLAllowed("https://10.0.0.1/go-gitea/gitea.git", nonAdminUser)
	require.NoError(t, err)

	old := setting.ImportLocalPaths
	setting.ImportLocalPaths = false

	_, err = IsMigrateURLAllowed("/home/foo/bar/goo", adminUser)
	require.Error(t, err)

	setting.ImportLocalPaths = true
	abs, err := filepath.Abs(".")
	require.NoError(t, err)

	_, err = IsMigrateURLAllowed(abs, adminUser)
	require.NoError(t, err)

	_, err = IsMigrateURLAllowed(abs, nonAdminUser)
	require.Error(t, err)

	nonAdminUser.AllowImportLocal = true
	_, err = IsMigrateURLAllowed(abs, nonAdminUser)
	require.NoError(t, err)

	setting.ImportLocalPaths = old
}

func TestURLAllowedSSH(t *testing.T) {
	require.NoError(t, unittest.PrepareTestDatabase())

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "user2"})
	sshURL := "ssh://git@git.gay/gitgay/forgejo"

	t.Run("Migrate URL", func(t *testing.T) {
		_, err := IsMigrateURLAllowed(sshURL, user)
		require.Error(t, err)
	})

	t.Run("Pushmirror URL", func(t *testing.T) {
		_, err := IsPushMirrorURLAllowed(sshURL, user)
		require.NoError(t, err)
	})
}
