// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTest(apiv1_permissions.ReqAnyRepoReader, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		{
			data: newTestData(map[string]string{}, newSharedData().
				SetDoer("root").
				SetDoerAdmin(true).
				SetRepositoryName("userowner/repositoryname").
				SetRepositoryPrivate(true),
			),
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
		// 	data: newTestData(map[string]string{}, newSharedData().
		// 		SetDoerName("doername").
		// 		SetRepositoryName("userowner/repositoryname").
		// 		SetRepositoryPrivate(true),
		// 	),
		// 	error: "Denied",
		// },
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"RepoAccess",
		"ReqAnyRepoReader",
	},
},
)
