// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package lfs

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"forgejo.org/models/perm"
	"forgejo.org/models/unittest"
	lfs_module "forgejo.org/modules/lfs"
	"forgejo.org/modules/private"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/services/contexttest"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const commonCfg = `[security]
INSTALL_LOCK = true
INTERNAL_TOKEN = ForgejoForgejoForgejoForgejoForgejoForgejo_	# don't use in prod
[oauth2]
JWT_SECRET = ForgejoForgejoForgejoForgejoForgejoForgejo_	# don't use in prod
[server]
LFS_START_SERVER = true
LFS_JWT_SECRET = ForgejoForgejoForgejoForgejoForgejoForgejo_
`

func TestMain(m *testing.M) {
	unittest.MainTest(m)
}

func getTokenString(t *testing.T, results *private.ServCommandResults, lfsVerb string) string {
	now := time.Now()
	claims := lfs_module.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(setting.LFS.HTTPAuthExpiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
		RepoID: results.RepoID,
		Op:     lfsVerb,
		UserID: results.UserID,
	}
	lfs := setting.LFS
	tokenString, err := lfs.SigningKey.JWT(claims)
	require.NoError(t, err)
	return tokenString
}

func TestCheckVersionCommand(t *testing.T) {
	s := SSHAdpater{}
	assert.False(t, s.checkVersionCommand([]byte("version 2\n")))
	assert.True(t, s.checkVersionCommand([]byte("version 1\n")))
}

func TestHandleLFSTransfer(t *testing.T) {
	lfsVerb := "upload"
	tests := []struct {
		name     string // description of test
		msg      string
		expected string
	}{
		{
			"Version check",
			strings.Join(
				[]string{
					"000eversion 1",
					"00000009quit",
					"0000",
				}, "\n",
			),
			strings.Join(
				[]string{
					"000eversion=1",
					"000clocking",
					"0000000fstatus 200",
					"0000000fstatus 200",
					"0000",
				}, "\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			setting.CfgProvider, err = setting.NewConfigProviderFromData(commonCfg)
			require.NoError(t, err, "Config")
			setting.LoadCommonSettings()
			defer test.MockVariableValue(&setting.LFS, setting.LFS)()
			setting.LFS.StartServer = true

			unittest.PrepareTestEnv(t)
			ctx, _ := contexttest.MockContext(t, "user2/repo1")
			ctx.SetParams(":id", "1")
			contexttest.LoadRepo(t, ctx, 1)
			contexttest.LoadUser(t, ctx, 2)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()

			requestedMode := perm.AccessModeWrite
			results := private.ServCommandResults{KeyID: 1, OwnerName: "user2", RepoName: "repo1", RepoID: 1, UserID: 2, UserName: "user2"}
			tokenString := getTokenString(t, &results, lfsVerb)

			var buf bytes.Buffer
			pktAdapter := NewPktAdapter(strings.NewReader(tt.msg), &buf)
			err = HandleLFSTransfer(ctx, &results, pktAdapter, requestedMode, lfsVerb, tokenString)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestHandleLFSTransferError(t *testing.T) {
	lfsVerb := "upload"
	tests := []struct {
		name     string // description of test
		msg      string
		expected string
		errorMsg string
	}{
		{
			"Version error",
			strings.Join(
				[]string{
					"000eversion 2",
					"00000009quit",
					"0000",
				}, "\n",
			),
			strings.Join(
				[]string{
					"000eversion=1",
					"000clocking",
					"0000000fstatus 400",
					"00010020Unexpected version received",
					"0000",
				}, "\n"),
			"Failed to match capability: \"version 2\\n\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			setting.CfgProvider, err = setting.NewConfigProviderFromData(commonCfg)
			require.NoError(t, err, "Config")
			setting.LoadCommonSettings()
			defer test.MockVariableValue(&setting.LFS, setting.LFS)()
			setting.LFS.StartServer = true

			unittest.PrepareTestEnv(t)
			ctx, _ := contexttest.MockContext(t, "user2/repo1")
			ctx.SetParams(":id", "1")
			contexttest.LoadRepo(t, ctx, 1)
			contexttest.LoadUser(t, ctx, 2)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()

			requestedMode := perm.AccessModeWrite
			results := private.ServCommandResults{KeyID: 1, OwnerName: "user2", RepoName: "repo1", RepoID: 1, UserID: 2, UserName: "user2"}
			tokenString := getTokenString(t, &results, lfsVerb)

			var buf bytes.Buffer
			pktAdapter := NewPktAdapter(strings.NewReader(tt.msg), &buf)
			err = HandleLFSTransfer(ctx, &results, pktAdapter, requestedMode, lfsVerb, tokenString)
			assert.Equal(t, tt.expected, buf.String())
			require.Error(t, err)
			assert.EqualError(t, err, tt.errorMsg)
		})
	}
}
