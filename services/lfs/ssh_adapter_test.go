package lfs

import (
	"bytes"
	"strings"
	"testing"

	"forgejo.org/models/perm"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/private"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCapability(t *testing.T) {
	s := SSHAdpater{}
	assert.Equal(t, "version=1\n", string(s.getCapabilityAdvertisement()))
}

func TestCheckVersionCommand(t *testing.T) {
	s := SSHAdpater{}
	assert.False(t, s.checkVersionCommand([]byte("version 2\n")))
	assert.True(t, s.checkVersionCommand([]byte("version 1\n")))
}

func TestHandleLFSTransfer(t *testing.T) {
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
					"0000000fstatus 200",
					"0000000fstatus 200",
					"0000",
				}, "\n"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unittest.PrepareTestEnv(t)
			ctx, _ := contexttest.MockContext(t, "user2/repo1")
			ctx.SetParams(":id", "1")
			contexttest.LoadRepo(t, ctx, 1)
			contexttest.LoadUser(t, ctx, 2)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()

			requestedMode := perm.AccessModeWrite
			results := private.ServCommandResults{KeyID: 1, OwnerName: "user2", RepoName: "repo1", RepoID: 1, UserID: 2, UserName: "user2"}
			var buf bytes.Buffer
			pktAdapter := NewPktAdapter(strings.NewReader(tt.msg), &buf)
			err := HandleLFSTransfer(ctx, &results, pktAdapter, requestedMode, "upload")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestHandleLFSTransferError(t *testing.T) {
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
					"0000000fstatus 400",
					"00010020Unexpected version received",
					"0000",
				}, "\n"),
			"Failed to match capability: \"version 2\\n\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unittest.PrepareTestEnv(t)
			ctx, _ := contexttest.MockContext(t, "user2/repo1")
			ctx.SetParams(":id", "1")
			contexttest.LoadRepo(t, ctx, 1)
			contexttest.LoadUser(t, ctx, 2)
			contexttest.LoadGitRepo(t, ctx)
			defer ctx.Repo.GitRepo.Close()

			requestedMode := perm.AccessModeWrite
			results := private.ServCommandResults{KeyID: 1, OwnerName: "user2", RepoName: "repo1", RepoID: 1, UserID: 2, UserName: "user2"}
			var buf bytes.Buffer
			pktAdapter := NewPktAdapter(strings.NewReader(tt.msg), &buf)
			err := HandleLFSTransfer(ctx, &results, pktAdapter, requestedMode, "upload")
			assert.Equal(t, tt.expected, buf.String())
			require.Error(t, err)
			assert.EqualError(t, err, tt.errorMsg)
		})
	}
}
