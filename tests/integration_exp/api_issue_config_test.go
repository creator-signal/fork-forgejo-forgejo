// Copyright 2023 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors c/o Codeberg e.V.. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"fmt"
	"testing"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	api "forgejo.org/modules/structs"
	"forgejo.org/tests/forgery"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func createIssueConfigInDirectory(t *testing.T, user *user_model.User, repo *repo_model.Repository, dir string, issueConfig map[string]any) {
	config, err := yaml.Marshal(issueConfig)
	require.NoError(t, err)

	err = createOrReplaceFileInBranch(user, repo, fmt.Sprintf("%s/ISSUE_TEMPLATE/config.yaml", dir), repo.DefaultBranch, string(config))
	require.NoError(t, err)
}

func createIssueConfig(t *testing.T, user *user_model.User, repo *repo_model.Repository, issueConfig map[string]any) {
	createIssueConfigInDirectory(t, user, repo, ".gitea", issueConfig)
}

func getIssueConfig(t *testing.T, sess forgery.Session, owner, repo string) api.IssueConfig {
	resp := sess.Get(t, "/api/v1/repos", owner, repo, "issue_config")
	return forgery.DecodeJSON[api.IssueConfig](t, resp)
}

func TestAPIRepoGetIssueConfig(t *testing.T) {
	t.Parallel()
	sess := forgery.SharedInstance(t).Session()

	owner := forgery.CreateUser(t, nil)
	repo, _, cleanup := forgery.CreateRepository(t, owner, nil)
	t.Cleanup(cleanup)

	t.Run("Default", func(t *testing.T) {
		issueConfig := getIssueConfig(t, sess, owner.Name, repo.Name)

		assert.True(t, issueConfig.BlankIssuesEnabled)
		assert.Empty(t, issueConfig.ContactLinks)
	})

	t.Run("DisableBlankIssues", func(t *testing.T) {
		config := make(map[string]any)
		config["blank_issues_enabled"] = false

		createIssueConfig(t, owner, repo, config)

		issueConfig := getIssueConfig(t, sess, owner.Name, repo.Name)

		assert.False(t, issueConfig.BlankIssuesEnabled)
		assert.Empty(t, issueConfig.ContactLinks)
	})

	t.Run("ContactLinks", func(t *testing.T) {
		contactLink := make(map[string]string)
		contactLink["name"] = "TestName"
		contactLink["url"] = "https://example.com"
		contactLink["about"] = "TestAbout"

		config := make(map[string]any)
		config["contact_links"] = []map[string]string{contactLink}

		createIssueConfig(t, owner, repo, config)

		issueConfig := getIssueConfig(t, sess, owner.Name, repo.Name)

		assert.True(t, issueConfig.BlankIssuesEnabled)
		assert.Len(t, issueConfig.ContactLinks, 1)

		assert.Equal(t, "TestName", issueConfig.ContactLinks[0].Name)
		assert.Equal(t, "https://example.com", issueConfig.ContactLinks[0].URL)
		assert.Equal(t, "TestAbout", issueConfig.ContactLinks[0].About)
	})

	t.Run("Full", func(t *testing.T) {
		contactLink := make(map[string]string)
		contactLink["name"] = "TestName"
		contactLink["url"] = "https://example.com"
		contactLink["about"] = "TestAbout"

		config := make(map[string]any)
		config["blank_issues_enabled"] = false
		config["contact_links"] = []map[string]string{contactLink}

		createIssueConfig(t, owner, repo, config)

		issueConfig := getIssueConfig(t, sess, owner.Name, repo.Name)

		assert.False(t, issueConfig.BlankIssuesEnabled)
		assert.Len(t, issueConfig.ContactLinks, 1)

		assert.Equal(t, "TestName", issueConfig.ContactLinks[0].Name)
		assert.Equal(t, "https://example.com", issueConfig.ContactLinks[0].URL)
		assert.Equal(t, "TestAbout", issueConfig.ContactLinks[0].About)
	})
}

func TestAPIRepoIssueConfigPaths(t *testing.T) {
	t.Parallel()
	sess := forgery.SharedInstance(t).Session()

	owner := forgery.CreateUser(t, nil)
	repo, _, cleanup := forgery.CreateRepository(t, owner, nil)
	t.Cleanup(cleanup)

	templateConfigCandidates := []string{
		".forgejo/ISSUE_TEMPLATE/config",
		".forgejo/issue_template/config",
		".gitea/ISSUE_TEMPLATE/config",
		".gitea/issue_template/config",
		".github/ISSUE_TEMPLATE/config",
		".github/issue_template/config",
		"docs/issue_template/config",
	}

	for _, candidate := range templateConfigCandidates {
		for _, extension := range []string{".yaml", ".yml"} {
			fullPath := candidate + extension
			t.Run(fullPath, func(t *testing.T) {
				configMap := make(map[string]any)
				configMap["blank_issues_enabled"] = false

				configData, err := yaml.Marshal(configMap)
				require.NoError(t, err)

				_, err = createFileInBranch(owner, repo, fullPath, repo.DefaultBranch, string(configData))
				require.NoError(t, err)

				issueConfig := getIssueConfig(t, sess, owner.Name, repo.Name)

				assert.False(t, issueConfig.BlankIssuesEnabled)
				assert.Empty(t, issueConfig.ContactLinks)

				err = deleteFileInBranch(owner, repo, fullPath, repo.DefaultBranch)
				require.NoError(t, err)
			})
		}
	}
}

func TestAPIRepoValidateIssueConfig(t *testing.T) {
	t.Parallel()
	sess := forgery.SharedInstance(t).Session()

	owner := forgery.CreateUser(t, nil)
	repo, _, cleanup := forgery.CreateRepository(t, owner, nil)
	t.Cleanup(cleanup)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issue_config/validate", owner.Name, repo.Name)

	t.Run("Valid", func(t *testing.T) {
		resp := sess.Get(t, urlStr)
		issueConfigValidation := forgery.DecodeJSON[api.IssueConfigValidation](t, resp)

		assert.True(t, issueConfigValidation.Valid)
		assert.Empty(t, issueConfigValidation.Message)
	})

	t.Run("Invalid", func(t *testing.T) {
		dirs := []string{".gitea", ".forgejo", "docs"}
		for _, dir := range dirs {
			t.Run(dir, func(t *testing.T) {
				defer func() {
					deleteFileInBranch(owner, repo, fmt.Sprintf("%s/ISSUE_TEMPLATE/config.yaml", dir), repo.DefaultBranch)
				}()

				config := make(map[string]any)
				config["blank_issues_enabled"] = "Test"

				createIssueConfigInDirectory(t, owner, repo, dir, config)

				resp := sess.Get(t, urlStr)
				issueConfigValidation := forgery.DecodeJSON[api.IssueConfigValidation](t, resp)

				assert.False(t, issueConfigValidation.Valid)
				assert.NotEmpty(t, issueConfigValidation.Message)
			})
		}
	})
}
