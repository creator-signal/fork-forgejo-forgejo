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
			data: newTestData(map[string]string{}, map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",
			}),
		},
		// This fixture is unreachable because this permissions function is always used after
		// a RepoAccess that enforces the same restriction for non admin users
		// {
		// 	data: newTestData(map[string]string{}, map[string]string{
		// 		"doer":        "doerregular",
		// 		"repository":  "userowner/repositoryprivate",
		// 		"issue":       "issueOne",
		// 		"issueAuthor": "issueAuthor",
		// 		"comment":     "comment for ReqValidCommentID",
		// 	}),
		// 	error: "Not Found",
		// },
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"NilIssue": "true",
			}),
			error: "Not Found",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"doer":        "doerregular",
				"repository":  "userowner/repositorypublic",
				"issue":       "issueOne",
				"issueAuthor": "issueAuthor",
				"comment":     "comment for ReqValidCommentID",

				"InconsistentID": "true",
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
		data.SetSharedDefault("issue", "issueOne")
		data.SetSharedDefault("issueAuthor", "issueAuthor")
		data.SetSharedDefault("comment", "comment for ReqValidCommentID")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		issueAuthor := fixtureCreateUser(t, &user_model.User{Name: data.GetShared("issueAuthor")})
		issue := fixtureSetIssue(t, permissions, data.GetShared("issue"), issueAuthor.Name)
		fixtureCreateComment(t, permissions, issue, data.GetShared("comment"))
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		comment := fixtureGetComment(t, data.GetShared("comment"))
		if data.HasShared("NilIssue") {
			comment.Issue = nil
		}
		if data.HasShared("InconsistentID") {
			comment.Issue.RepoID = 123456
		}
		t.Logf("calling ReqValidCommentID(ctx, %+v)", comment)
		apiv1_permissions.ReqValidCommentID(ctx, comment)
	},
})
