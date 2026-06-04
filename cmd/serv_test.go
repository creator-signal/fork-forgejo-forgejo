package cmd

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"forgejo.org/modules/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func captureIO(t *testing.T) (cleanup func(), finish func() (output string), inWrite *os.File) {
	t.Helper()

	inRead, inWrite, err := os.Pipe()
	if err != nil {
		t.Errorf("captureIo cannot create stdout pipe: %v", err)
	}
	savedIn := os.Stdin
	os.Stdin = inRead

	outRead, outWrite, err := os.Pipe()
	if err != nil {
		t.Errorf("captureIo cannot create stdout pipe: %v", err)
	}
	savedOut := os.Stdout
	os.Stdout = outWrite

	return func() {
			outWrite.Close()
			os.Stdout = savedOut
			inRead.Close()
			inWrite.Close()
			os.Stdin = savedIn
		}, func() (output string) {
			out, err := io.ReadAll(outRead)
			if err != nil {
				t.Errorf("captureIo cannot read stdout pipe: %v", err)
			}
			return string(out)
		}, inWrite
}

func TestLfsTransfer_VersionNegotiation(t *testing.T) {
	msg := strings.Join(
		[]string{
			"000eversion 1",
			"00000009quit",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"0000000fstatus 200",
			"0000000fstatus 200",
			"0000",
		}, "\n",
	)

	app := cmdServ()
	app.Before = nil
	app.ReadArgsFromStdin = false
	app.ExitErrHandler = func(_ context.Context, _ *cli.Command, err error) {
		require.NoError(t, err)
	}

	t.Setenv("SSH_ORIGINAL_COMMAND", "git-lfs-transfer user40/repo60.git download")
	setting.IsProd = false
	setting.LFS.StartServer = true

	cleanup, finish, inWrite := captureIO(t)
	go func() {
		time.Sleep(2 * time.Second)
		cleanup()
	}()
	defer cleanup()
	inWrite.WriteString(msg)

	err := app.Run(t.Context(), []string{"test-serv", "key-1", "--debug"})
	out := finish()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestLfsTransfer_VersionError(t *testing.T) {
	msg := strings.Join(
		[]string{
			"000eversion 2",
			"0000",
		}, "\n",
	)
	expected := strings.Join(
		[]string{
			"000eversion=1",
			"0000000fstatus 400",
			"00010020Unexpected version received",
			"0000",
		}, "\n",
	)

	app := cmdServ()
	app.Before = nil
	app.ReadArgsFromStdin = false
	app.ExitErrHandler = func(_ context.Context, _ *cli.Command, err error) {
		assert.Error(t, err)
	}

	t.Setenv("SSH_ORIGINAL_COMMAND", "git-lfs-transfer user40/repo60.git download")
	setting.IsProd = false
	setting.LFS.StartServer = true

	cleanup, finish, inWrite := captureIO(t)
	go func() {
		time.Sleep(2 * time.Second)
		cleanup()
	}()
	defer cleanup()
	inWrite.WriteString(msg)

	err := app.Run(t.Context(), []string{"test-serv", "key-1", "--debug"})
	out := finish()
	require.Error(t, err)
	assert.Equal(t, expected, out)
}
