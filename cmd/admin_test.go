package cmd

import (
	"context"
	"errors"
	"testing"

	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/modules/translation"
	"forgejo.org/services/cron"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestExecTask(t *testing.T) {

	calledTest1 := false
	calledTest2 := false

	// Setup the environment.
	defer test.MockVariableValue(&setting.InstallLock, true)()
	translation.InitLocales(t.Context())

	// Reusing existing tasknames for this, since
	cron.RegisterTaskFatal(cron.TaskArchiveCleanup, &cron.BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ cron.Config) error {
		calledTest1 = true
		return nil
	})

	cron.RegisterTaskFatal(cron.TaskCancelAbandonedJobs, &cron.BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   "@every 1h",
	}, func(ctx context.Context, _ *user_model.User, _ cron.Config) error {
		calledTest2 = true
		return errors.New("This should not be called")
	})

	app := newTestApp(func(_ context.Context, ctx *cli.Command) error { return nil })
	err := RunMainApp(app, "./gitea", "admin", "exectask", "--taskname", cron.TaskArchiveCleanup)
	require.NoError(t, err)
	require.True(t, calledTest1)
	require.False(t, calledTest2)

}
