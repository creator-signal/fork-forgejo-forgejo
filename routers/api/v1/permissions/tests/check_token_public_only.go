// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package tests

import (
	"fmt"
	"strings"
	"testing"

	auth_model "forgejo.org/models/auth"
	org_model "forgejo.org/models/organization"
	user_model "forgejo.org/models/user"
	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
)

const (
	categoryActivityPub  = "activitypub"
	categoryAdmin        = "admin"
	categoryNotification = "notification"
	categoryOrganization = "organization"
	categoryPackage      = "package"
	categoryIssue        = "issue"
	categoryRepository   = "repository"
	categoryUser         = "user"
)

var categoryStringToCategory = map[string]auth_model.AccessTokenScopeCategory{
	categoryActivityPub:  auth_model.AccessTokenScopeCategoryActivityPub,
	categoryAdmin:        auth_model.AccessTokenScopeCategoryAdmin,
	categoryNotification: auth_model.AccessTokenScopeCategoryNotification,
	categoryOrganization: auth_model.AccessTokenScopeCategoryOrganization,
	categoryPackage:      auth_model.AccessTokenScopeCategoryPackage,
	categoryIssue:        auth_model.AccessTokenScopeCategoryIssue,
	categoryRepository:   auth_model.AccessTokenScopeCategoryRepository,
	categoryUser:         auth_model.AccessTokenScopeCategoryUser,
}

var _ = registerFunctionTestWithCall(apiv1_permissions.CheckTokenPublicOnly, functionTest{
	testCases: []*testCase{
		{
			data: newTestData(map[string]string{}, map[string]string{}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository": "userowner/repositorypublic",
				"scope":      fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositorypublic",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryRepository,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositoryprivate",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryRepository,
			}),
			error: "token scope is limited to public repos",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositorypublic",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryIssue,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositoryprivate",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryIssue,
			}),
			error: "token scope is limited to public issues",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositorypublic",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryNotification,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"repository":              "userowner/repositoryprivate",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryNotification,
			}),
			error: "token scope is limited to public notifications",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"user":                    "regularuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryUser,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"user":                    "privateuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryUser,
			}),
			error: "token scope is limited to public users",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"user":                    "regularuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryActivityPub,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"user":                    "privateuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryActivityPub,
			}),
			error: "token scope is limited to public activitypub",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":                     "regularorg",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryOrganization,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":                     "privateorg",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryOrganization,
			}),
			error: "token scope is limited to public orgs",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"org":                     "privateorg",
				"orgAsUser":               "true",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryOrganization,
			}),
			error: "token scope is limited to public orgs",
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"packageOwner":            "regularuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryPackage,
			}),
		},
		{
			data: newTestData(map[string]string{}, map[string]string{
				"packageOwner":            "privateuser",
				"scope":                   fmt.Sprintf("%s", auth_model.AccessTokenScopePublicOnly),
				"requiredScopeCategories": categoryPackage,
			}),
			error: "token scope is limited to public packages",
		},
	},
	sequenceFilter: []string{
		"APIAuthorization",
		"CheckTokenPublicOnly",
	},
	fulfillNeeds: func(t *testing.T, data *testData) {
		data.SetSharedDefault("doer", "regularuser")
		data.SetSharedDefault("repository", "regularuser/repositorypublic")
	},
	interpret: func(t *testing.T, permissions *apiv1_permissions.Permissions, data *testData) {
		if data.HasShared("user") {
			fixtureCreateUser(t, &user_model.User{Name: data.GetShared("user")})
		}
		if data.HasShared("org") {
			fixtureCreateOrg(t, &org_model.Organization{Name: data.GetShared("org")}, &user_model.User{Name: data.GetShared("doer")})
		}
		if data.HasShared("packageOwner") {
			fixtureCreateUser(t, &user_model.User{Name: data.GetShared("packageOwner")})
		}
		if data.HasShared("requiredScopeCategories") {
			var categories []auth_model.AccessTokenScopeCategory
			for categoryString := range strings.SplitSeq(data.GetShared("requiredScopeCategories"), ",") {
				categories = append(categories, categoryStringToCategory[categoryString])
			}
			permissions.SetRequiredScopeCategories(categories)
		}
		fixtureSetRepository(t, permissions, data)
	},
	call: func(t *testing.T, ctx apiv1_permissions.Context, data *testData, _ []any) {
		t.Helper()
		var user *user_model.User
		if data.HasShared("user") {
			user = fixtureGetUser(t, data.GetShared("user"))
		}
		var org *org_model.Organization
		if data.HasShared("org") {
			if data.HasShared("orgAsUser") {
				user = fixtureGetUser(t, data.GetShared("org"))
			} else {
				org = fixtureGetOrg(t, data.GetShared("org"))
			}
		}
		var packageOwner *user_model.User
		if data.HasShared("packageOwner") {
			packageOwner = fixtureGetUser(t, data.GetShared("packageOwner"))
		}
		t.Logf("calling CheckTokenPublicOnly(ctx, %+v, %+v, %+v)", user, org, packageOwner)
		apiv1_permissions.CheckTokenPublicOnly(ctx, user, org.AsUser(), packageOwner)
	},
})
