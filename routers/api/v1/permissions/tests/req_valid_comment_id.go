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
			}, map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
		// 	data: newTestData(map[string]string{
		// 		"issue":       "issueOne",
		// 		"issueAuthor": "issueAuthor",
		// 		"comment":     "comment for ReqValidCommentID",
		// 	}, map[string]string{
		// 		"doer":        "doerregular",
		// 		"repository":  "userowner/repositoryprivate",
		// 	}),
		// 	error: "Not Found",
		// },
		{
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"NilIssue": "true",
			}, map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"InconsistentID": "true",
			}, map[string]string{
				"doer":       "doerregular",
				"repository": "userowner/repositorypublic",
			}),
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
		data.SetOwnDefault("issue", "issueOne")
		data.SetOwnDefault("issueAuthor", "issueAuthor")
		data.SetOwnDefault("comment", "comment for ReqValidCommentID")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.GetOwn("issueAuthor")})
		issue := fixtureSetIssue(t, permissions, data.GetOwn("issue"), issueAuthor.Name)
		fixtureCreateComment(t, permissions, issue, data.GetOwn("comment"))
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		comment := fixtureGetComment(t, data.GetOwn("comment"))
		if data.HasOwn("NilIssue") {
			comment.Issue = nil
		}
		if data.HasOwn("InconsistentID") {
			comment.Issue.RepoID = 123456
		}
		t.Logf("calling ReqValidCommentID(ctx, %+v)", comment)
		apiv1_permissions.ReqValidCommentID(ctx, comment)
	},
})
