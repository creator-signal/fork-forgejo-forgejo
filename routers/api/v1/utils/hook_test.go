// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package utils

import (
	"net/http"
	"reflect"
	"testing"

	"forgejo.org/models/unittest"
	"forgejo.org/modules/structs"
	webhook_module "forgejo.org/modules/webhook"
	"forgejo.org/services/contexttest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestHookValidation(t *testing.T) {
	unittest.PrepareTestEnv(t)

	t.Run("Test Validation", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		checkCreateHookOption(ctx, &structs.CreateHookOption{
			Type: "gitea",
			Config: map[string]string{
				"content_type": "json",
				"url":          "https://example.com/webhook",
			},
		})
		assert.Equal(t, 0, ctx.Resp.WrittenStatus()) // not written yet
	})

	t.Run("Test Validation with invalid URL", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		checkCreateHookOption(ctx, &structs.CreateHookOption{
			Type: "gitea",
			Config: map[string]string{
				"content_type": "json",
				"url":          "example.com/webhook",
			},
		})
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})

	t.Run("Test Validation with invalid webhook type", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		checkCreateHookOption(ctx, &structs.CreateHookOption{
			Type: "unknown",
			Config: map[string]string{
				"content_type": "json",
				"url":          "example.com/webhook",
			},
		})
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})

	t.Run("Test Validation with empty content type", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		checkCreateHookOption(ctx, &structs.CreateHookOption{
			Type: "unknown",
			Config: map[string]string{
				"url": "https://example.com/webhook",
			},
		})
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})
}

func TestHookEventInclusion(t *testing.T) {
	ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
	contexttest.LoadRepo(t, ctx, 1)
	contexttest.LoadGitRepo(t, ctx)
	contexttest.LoadRepoCommit(t, ctx)
	contexttest.LoadUser(t, ctx, 2)

	opts := structs.CreateHookOption{
		Type: "forgejo",
		Config: structs.CreateHookOptionConfig{
			"content_type": "json",
			"url":          "http://example.com/webhook",
		},
		Events: []string{
			string(webhook_module.HookEventCreate),
			string(webhook_module.HookEventDelete),
			string(webhook_module.HookEventFork),
			string(webhook_module.HookEventIssues),
			string(webhook_module.HookEventPush),
			string(webhook_module.HookEventPullRequest),
			string(webhook_module.HookEventWiki),
			string(webhook_module.HookEventRepository),
			string(webhook_module.HookEventRelease),
			string(webhook_module.HookEventPackage),
			string(webhook_module.HookEventActionRunFailure),
			string(webhook_module.HookEventActionRunRecover),
			string(webhook_module.HookEventActionRunSuccess),
		},
	}
	hook, ok := addHook(ctx, &opts, 2, 1)
	require.True(t, ok)
	val := reflect.ValueOf(hook.HookEvents)
	ty := val.Type()
	for i := range val.NumField() {
		assert.Truef(t, val.Field(i).Interface().(bool), "missing '%s' event", ty.Field(i).Name)
	}
}

func TestPackagistWebhookCreation(t *testing.T) {
	unittest.PrepareTestEnv(t)

	t.Run("Success with all required fields", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		opts := &structs.CreateHookOption{
			Type: string(webhook_module.PACKAGIST),
			Config: map[string]string{
				"url":          "https://packagist.org/api/update-package?username=testuser&apiToken=testtoken",
				"content_type": "json",
				"username":     "testuser",
				"api_token":    "testtoken123",
				"package_url":  "https://packagist.org/packages/test/package",
			},
		}

		hook, ok := addHook(ctx, opts, 2, 1)
		require.True(t, ok)
		require.NotNil(t, hook)
		assert.NotEmpty(t, hook.Meta)
		assert.Contains(t, hook.Meta, "testuser")
		assert.Contains(t, hook.Meta, "testtoken123")
		assert.Contains(t, hook.Meta, "https://packagist.org/packages/test/package")
	})

	t.Run("Missing username", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		opts := &structs.CreateHookOption{
			Type: string(webhook_module.PACKAGIST),
			Config: map[string]string{
				"url":          "https://packagist.org/api/update-package",
				"content_type": "json",
				"api_token":    "testtoken123",
				"package_url":  "https://packagist.org/packages/test/package",
			},
		}

		_, ok := addHook(ctx, opts, 2, 1)
		assert.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})

	t.Run("Missing api_token", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		opts := &structs.CreateHookOption{
			Type: string(webhook_module.PACKAGIST),
			Config: map[string]string{
				"url":          "https://packagist.org/api/update-package",
				"content_type": "json",
				"username":     "testuser",
				"package_url":  "https://packagist.org/packages/test/package",
			},
		}

		_, ok := addHook(ctx, opts, 2, 1)
		assert.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})

	t.Run("Missing package_url", func(t *testing.T) {
		ctx, _ := contexttest.MockAPIContext(t, "user2/repo1/hooks")
		contexttest.LoadRepo(t, ctx, 1)
		contexttest.LoadGitRepo(t, ctx)
		contexttest.LoadRepoCommit(t, ctx)
		contexttest.LoadUser(t, ctx, 2)

		opts := &structs.CreateHookOption{
			Type: string(webhook_module.PACKAGIST),
			Config: map[string]string{
				"url":          "https://packagist.org/api/update-package",
				"content_type": "json",
				"username":     "testuser",
				"api_token":    "testtoken123",
			},
		}

		_, ok := addHook(ctx, opts, 2, 1)
		assert.False(t, ok)
		assert.Equal(t, http.StatusUnprocessableEntity, ctx.Resp.WrittenStatus())
	})
}
