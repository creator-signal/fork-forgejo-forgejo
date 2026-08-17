// Copyright Forgejo Authors
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"context"
	"net/url"
	"testing"

	f3_forge_model "forgejo.org/models/f3/forge"
	f3_mirror_model "forgejo.org/models/f3/mirror"
	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	apiv1 "forgejo.org/routers/api/v1"
	f3_context "forgejo.org/services/f3/context"
	f3_driver "forgejo.org/services/f3/driver"
	driver_options "forgejo.org/services/f3/driver/options"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/perm/access"
	_ "forgejo.org/services/f3/driver/tests"

	tests_f3 "code.forgejo.org/f3/gof3/v3/tree/tests/f3"
	"github.com/stretchr/testify/require"
)

func TestServicesF3(t *testing.T) {
	defer test.MockVariableValue(&f3_driver.PullRequestAddToQueue, func(ctx context.Context, pr *issues_model.PullRequest) {})()
	defer test.MockVariableValue(&setting.SSH.RootPath, t.TempDir())()
	defer test.MockVariableValue(&setting.DisableGitHooks, false)()

	// because setting.IsInTesting == true, it will record the
	// middleware sequence of each route it builds
	apiv1.Routes()

	// the server will be used when obtaining avatars
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		log.SetConsoleLogger(log.DEFAULT, "console", log.TRACE)
		defer func() {
			log.SetConsoleLogger(log.DEFAULT, "console", log.INFO)
		}()
		forge := f3_forge_model.NewForge()
		forge.SetURL("URL")
		forge, err := f3_forge_model.Upsert(t.Context(), forge)
		require.NoError(t, err)
		mirror := f3_mirror_model.NewMirror()
		mirror.SetForgeID(forge.ID)
		mirror, err = f3_mirror_model.Upsert(t.Context(), mirror)
		require.NoError(t, err)
		tests_f3.ForgeCompliance(f3_context.WithF3(t.Context(), f3_context.New().SetMirrorID(mirror.ID)), t, driver_options.Name)
	})
}
