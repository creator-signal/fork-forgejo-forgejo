// Copyright Forgejo Authors
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"context"
	"net/url"
	"testing"

	issues_model "forgejo.org/models/issues"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	f3_context "forgejo.org/services/f3/context"
	f3_driver "forgejo.org/services/f3/driver"
	driver_options "forgejo.org/services/f3/driver/options"

	_ "forgejo.org/models"
	_ "forgejo.org/models/actions"
	_ "forgejo.org/models/activities"
	_ "forgejo.org/models/perm/access"
	_ "forgejo.org/services/f3/driver/tests"

	tests_f3 "code.forgejo.org/f3/gof3/v3/tree/tests/f3"
)

func TestServicesF3(t *testing.T) {
	defer test.MockVariableValue(&f3_driver.PullRequestAddToQueue, func(ctx context.Context, pr *issues_model.PullRequest) {})()
	defer test.MockVariableValue(&setting.SSH.RootPath, t.TempDir())()
	// the server will be used when obtaining avatars
	onApplicationRun(t, func(t *testing.T, u *url.URL) {
		log.SetConsoleLogger(log.DEFAULT, "console", log.TRACE)
		defer func() {
			log.SetConsoleLogger(log.DEFAULT, "console", log.INFO)
		}()
		tests_f3.ForgeCompliance(f3_context.WithF3(t.Context()), t, driver_options.Name)
	})
}
