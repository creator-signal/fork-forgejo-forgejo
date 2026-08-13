// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package integration

import (
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	apiv1 "forgejo.org/routers/api/v1"
	funding_tests "forgejo.org/services/funding/tests"
)

func TestFundingRetrieval(t *testing.T) {
	defer test.MockVariableValue(&setting.Service.DefaultAllowCreateOrganization, true)()
	defer test.MockVariableValue(&setting.IsInTesting, true)()
	defer test.MockVariableValue(&setting.DisableGitHooks, false)()

	// defer tests.PrepareTestEnv(t)() // this didn't work either 😞
	unittest.PrepareTestEnv(t)

	// because setting.IsInTesting == true, it will record the
	// middleware sequence of each route it builds
	apiv1.Routes()

	funding_tests.FromDefaultBranch(t)
}
