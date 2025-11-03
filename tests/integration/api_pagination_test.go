// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

// Every endpoint with `page` & `limit` parameters, and `X-Total-Count` and
// `Link` response headers is automatically checked here.
//
// If an endpoint has path parameters, add them to the `params` map. There are
// helper functions `argsRepo`, `argsIssue`, and `argsPullRequest` that create
// the necessary parameters from the item's ID. If you need to call the endpoint
// as another user, add a "Sudo" parameter to that endpoint's params map, set to
// the name of the user to execute as.
//
// If you need new fixtures to test an endpoint, add them into:
//   tests/integration/fixtures/pagination/{OPERATION_ID}
// It will automatically load them in the test.
//
// If you need to do other miscellanous setup, add a function to the `setup`
// map. The function it returns is `defer`red to the end of the test.
//
// If this framework doesn't work for testing an endpoint, add it to the `skip`
// map and make a test for it elsewhere.

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	"forgejo.org/models/auth"
	issue_model "forgejo.org/models/issues"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/models/unittest"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	swagger_spec "github.com/go-openapi/spec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Returns the path arguments needed to refer to the repo with the given ID
func argsRepo(t *testing.T, id int64) map[string]string {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: id})
	return map[string]string{
		"owner": repo.OwnerName,
		"repo":  repo.Name,
	}
}

// Returns the path arguments needed to refer to the issue with the given ID
func argsIssue(t *testing.T, id int64) map[string]string {
	issue := unittest.AssertExistsAndLoadBean(t, &issue_model.Issue{ID: id})
	args := argsRepo(t, issue.RepoID)
	args["index"] = fmt.Sprint(issue.Index)
	return args
}

// Returns the path arguments needed to refer to the issue with the given ID
func argsPullRequest(t *testing.T, id int64) map[string]string {
	pull := unittest.AssertExistsAndLoadBean(t, &issue_model.PullRequest{ID: id})
	args := argsRepo(t, pull.BaseRepoID)
	args["index"] = fmt.Sprint(pull.Index)
	return args
}

func merge(a, b map[string]string) map[string]string {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func TestAPIPaginatedResponses(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	swagger := getSwagger(t)

	params := map[string]map[string]string{
		"getOrgVariablesList":          {"org": "org3"},
		"getRepoVariablesList":         argsRepo(t, 1),
		"GetTree":                      merge(argsRepo(t, 1), map[string]string{"sha": "62fb502a7172d4453f0322a2cc85bddffa57f07a"}),
		"issueGetCommentsAndTimeline":  argsIssue(t, 1),
		"issueGetIssueReactions":       argsIssue(t, 1),
		"issueGetMilestonesList":       argsRepo(t, 1),
		"issueGetRepoComments":         argsRepo(t, 1),
		"issueListBlocks":              argsIssue(t, 1),
		"issueListIssueDependencies":   merge(argsRepo(t, 6), map[string]string{"index": "1"}),
		"issueListIssues":              argsRepo(t, 1),
		"issueListLabels":              argsIssue(t, 2),
		"issueSubscriptions":           argsIssue(t, 7),
		"issueTrackedTimes":            argsIssue(t, 2),
		"ListActionRuns":               argsRepo(t, 4),
		"ListActionTasks":              argsRepo(t, 4),
		"listForks":                    {"owner": "user2", "repo": "repo65"},
		"listPackages":                 {"owner": "user1"},
		"notifyGetList":                {"Sudo": "user2"},
		"notifyGetRepoList":            merge(argsRepo(t, 1), map[string]string{"Sudo": "user2"}),
		"orgListActionsSecrets":        {"org": "org3"},
		"orgListActivityFeeds":         {"org": "org3"},
		"orgListBlockedUsers":          {"org": "org3"},
		"orgListHooks":                 {"org": "org3"},
		"orgListLabels":                {"org": "org3"},
		"orgListMembers":               {"org": "org3"},
		"orgListPublicMembers":         {"org": "org3"},
		"orgListQuotaArtifacts":        {"org": "org3"},
		"orgListQuotaAttachments":      {"org": "org3"},
		"orgListQuotaPackages":         {"org": "org3"},
		"orgListRepos":                 {"org": "org3"},
		"orgListTeamActivityFeeds":     {"id": "1"},
		"orgListTeamMembers":           {"id": "2"},
		"orgListTeamRepos":             {"id": "1"},
		"orgListTeams":                 {"org": "org3"},
		"orgListUserOrgs":              {"username": "user1"},
		"repoGetAllCommits":            argsRepo(t, 2),
		"repoGetCombinedStatusByRef":   merge(argsRepo(t, 62), map[string]string{"ref": "774f93df12d14931ea93259ae93418da4482fcc1"}),
		"repoGetPullRequestCommits":    argsPullRequest(t, 2),
		"repoGetPullRequestFiles":      argsPullRequest(t, 2),
		"repoGetWikiPages":             argsRepo(t, 1),
		"repoListActionsSecrets":       argsRepo(t, 1),
		"repoListActivityFeeds":        merge(argsRepo(t, 9), map[string]string{"Sudo": "user11"}),
		"repoListBranches":             argsRepo(t, 1),
		"repoListCollaborators":        argsRepo(t, 4),
		"repoListHooks":                argsRepo(t, 1),
		"repoListKeys":                 argsRepo(t, 1),
		"repoListPullRequests":         argsRepo(t, 1),
		"repoListPullReviews":          argsIssue(t, 2),
		"repoListPushMirrors":          argsRepo(t, 1),
		"repoListReleases":             argsRepo(t, 1),
		"repoListStargazers":           {"owner": "user2", "repo": "repo64"}, // can't use argsRepo for repos defined in a custom fixture
		"repoListStatuses":             merge(argsRepo(t, 62), map[string]string{"sha": "774f93df12d14931ea93259ae93418da4482fcc1"}),
		"repoListStatusesByRef":        merge(argsRepo(t, 62), map[string]string{"ref": "774f93df12d14931ea93259ae93418da4482fcc1"}),
		"repoListSubscribers":          argsRepo(t, 1),
		"repoListTags":                 argsRepo(t, 57),
		"repoListTopics":               argsRepo(t, 1),
		"repoTrackedTimes":             argsRepo(t, 1),
		"teamSearch":                   {"org": "org3"},
		"userCurrentListFollowers":     {"Sudo": "user8"},
		"userCurrentListFollowing":     {"Sudo": "user4"},
		"userCurrentListGPGKeys":       {"Sudo": "user2"},
		"userCurrentListKeys":          {"Sudo": "user2"},
		"userCurrentListRepos":         {"Sudo": "user2"},
		"userCurrentListStarred":       {"Sudo": "user2"},
		"userCurrentListSubscriptions": {"Sudo": "user4"},
		"userGetTokens":                {"username": "user1"},
		"userListActivityFeeds":        {"username": "user10"},
		"userListBlockedUsers":         {"Sudo": "user4"},
		"userListFollowers":            {"username": "user8"},
		"userListFollowing":            {"username": "user4"},
		"userListGPGKeys":              {"username": "user2"},
		"userListKeys":                 {"username": "user2"},
		"userListQuotaArtifacts":       {"Sudo": "user5"},
		"userListQuotaAttachments":     {"Sudo": "user2"},
		"userListQuotaPackages":        {"Sudo": "user2"},
		"userListRepos":                {"username": "user2"},
		"userListStarred":              {"username": "user2"},
		"userListSubscriptions":        {"username": "user4", "Sudo": "user4"}, // For some reason this doesn't work when executed as user1
	}

	enableQuota := func(t *testing.T) func() {
		cleanupSetting := test.MockVariableValue(&setting.Quota.Enabled, true)
		cleanupRoute := test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())
		return func() {
			cleanupRoute()
			cleanupSetting()
		}
	}

	setup := map[string]func(*testing.T) func(){
		"userListQuotaPackages":    enableQuota,
		"userListQuotaArtifacts":   enableQuota,
		"userListQuotaAttachments": enableQuota,
		"orgListQuotaPackages":     enableQuota,
		"orgListQuotaAttachments":  enableQuota,
		"orgListQuotaArtifacts":    enableQuota,
	}

	// Only add endpoints to this list if they're being tested elsewhere!
	// Include the name of the test
	skip := map[string]struct{}{
		// TestListUnadoptedRepositories_ListOptions
		"adminUnadoptedList": {},
	}

	for path, pathData := range swagger.Paths.Paths {
		if pathData.Get == nil || pathData.Get.Deprecated {
			continue
		}
		if _, ok := skip[pathData.Get.ID]; ok {
			continue
		}

		var hasPageParam bool
		var hasLimitParam bool
		for i := range pathData.Get.Parameters {
			parameter := pathData.Get.Parameters[i]
			if parameter.Name == "page" {
				hasPageParam = true
			} else if parameter.Name == "limit" {
				hasLimitParam = true
			}
		}

		var hasTotalCountHeader bool
		var hasLinkHeader bool
		for responseCode, responseMaybe := range pathData.Get.Responses.StatusCodeResponses {
			var response swagger_spec.Response
			if len(responseMaybe.Ref.String()) > 0 {
				res, err := swagger_spec.ResolveResponse(swagger, responseMaybe.Ref)
				if err != nil {
					t.Errorf("failed to resolve response reference [%s]", err.Error())
				}
				response = *res
			} else {
				response = responseMaybe
			}
			if responseCode < 200 || responseCode >= 300 {
				continue
			}
			for headerName := range response.Headers {
				if headerName == "X-Total-Count" {
					hasTotalCountHeader = true
				} else if headerName == "Link" {
					hasLinkHeader = true
				}
			}
		}

		isPaginated := hasPageParam && hasLimitParam && hasLinkHeader && hasTotalCountHeader
		if !isPaginated {
			continue
		}

		t.Run(pathData.Get.ID, func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()
			fixturePath := "tests/integration/fixtures/pagination/" + pathData.Get.ID
			if _, err := os.Stat(fixturePath); err == nil {
				t.Log("Found custom fixtures")
				defer unittest.OverrideFixtures(fixturePath)()
			}
			if setupFunc, ok := setup[pathData.Get.ID]; ok {
				defer setupFunc(t)()
			}
			require.NoError(t, unittest.PrepareTestDatabase())

			urlPath := path
			paramValues := params[pathData.Get.ID]
			anyFailed := false
			for i := range pathData.Get.Parameters {
				parameter := pathData.Get.Parameters[i]
				if parameter.In == "path" {
					value, ok := paramValues[parameter.Name]
					if !ok {
						t.Errorf("no value set for %s parameter", parameter.Name)
						anyFailed = true
					}
					urlPath = strings.Replace(urlPath, "{"+parameter.Name+"}", value, 1)
				}
			}

			var actingUser string
			if sudo, ok := paramValues["Sudo"]; ok {
				actingUser = sudo
			} else {
				actingUser = "user1"
			}
			token := getUserToken(t, actingUser, auth.AccessTokenScopeAll)
			if anyFailed {
				t.FailNow()
			}

			urlPath = "/api/v1" + urlPath
			resp := MakeRequest(t, NewRequest(t, "GET", urlPath+"?page=2&limit=1").AddTokenAuth(token), http.StatusOK)
			if resp.Code != http.StatusOK {
				t.Fatalf("request failed: %s", resp.Body)
			}

			totalCountString := resp.Header().Get("X-Total-Count")
			assert.NotEmpty(t, totalCountString, "no X-Total-Count header in response")
			total, err := strconv.ParseInt(totalCountString, 10, 64)
			require.NoErrorf(t, err, "failed to parse X-Total-Count header: %s", err)
			assert.GreaterOrEqualf(t, total, int64(2), "not enough fixture items to test pagination (got %d, need at least 2), please add some to 'tests/integration/fixtures/pagination/%s`", total, pathData.Get.ID)

			linkHeader := resp.Header().Get("Link")
			assert.NotEmpty(t, linkHeader, "no Link header in response")
			links := strings.Split(linkHeader, ",")
			assert.Containsf(t, links, "<http://localhost:3003"+urlPath+"?limit=1&page=1>; rel=\"first\"", "Link header does not contain `first` link")
		})
	}
}
