// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"testing"

	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

var _ = registerFunctionTestWithCall(apiv1_permissions.ReqValidCommentID, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",
			}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
		// 	data: newTestData(map[string]string{
		// 		"issue":       "issueOne",
		// 		"issueAuthor": "issueAuthor",
		// 		"comment":     "comment for ReqValidCommentID",
		// 	}, newSharedData().
		// 		SetDoerName("doername").
		// 		SetRepositoryName("userowner/repositoryprivate").
		// 		SetRepositoryPrivate(true),
		// 	),
		// 	error: "Not Found",
		// },
		{
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"NilIssue": "true",
			}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"InconsistentID": "true",
			}, newSharedData().
				SetDoer("doername").
				SetRepositoryName("userowner/repositorypublic"),
			),
			error: "Not Found",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"RepoAccess",
		"ReqValidCommentID",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		t.Helper()
		data.SetDefault("issue", "issueOne")
		data.SetDefault("issueAuthor", "issueAuthor")
		data.SetDefault("comment", "comment for ReqValidCommentID")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.Get("issueAuthor")})
		issue := fixtureSetIssue(t, permissions, data.Get("issue"), issueAuthor.Name)
		fixtureCreateComment(t, permissions, issue, data.Get("comment"))
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		comment := fixtureGetComment(t, data.Get("comment"))
		if data.Has("NilIssue") {
			comment.Issue = nil
		}
		if data.Has("InconsistentID") {
			comment.Issue.RepoID = 123456
		}
		t.Logf("calling ReqValidCommentID(ctx, %+v)", comment)
		apiv1_permissions.ReqValidCommentID(ctx, comment)
	},
})
