// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.MustNotBeArchived, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetRepository(),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetRepository().
				SetRepositoryArchived(true),
			),
			error: "is archived",
		},
	},
})
