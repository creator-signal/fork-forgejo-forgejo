// Copyright 2020 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package cron

import (
	"context"
	"time"

	"forgejo.org/models"
	git_model "forgejo.org/models/git"
	user_model "forgejo.org/models/user"
	"forgejo.org/models/webhook"
	"forgejo.org/modules/git"
	"forgejo.org/modules/setting"
	"forgejo.org/services/auth"
	"forgejo.org/services/migrations"
	mirror_service "forgejo.org/services/mirror"
	packages_cleanup_service "forgejo.org/services/packages/cleanup"
	repo_service "forgejo.org/services/repository"
	archiver_service "forgejo.org/services/repository/archiver"
)

func registerUpdateMirrorTask() {
	type UpdateMirrorTaskConfig struct {
		BaseConfig
		PullLimit int
		PushLimit int
	}

	RegisterTaskFatal(TaskUpdateMirrors, &UpdateMirrorTaskConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: false,
			Schedule:   "@every 10m",
		},
		PullLimit: 50,
		PushLimit: 50,
	}, func(ctx context.Context, _ *user_model.User, cfg Config) error {
		umtc := cfg.(*UpdateMirrorTaskConfig)
		return mirror_service.Update(ctx, umtc.PullLimit, umtc.PushLimit)
	})
}

func registerRepoHealthCheck() {
	type RepoHealthCheckConfig struct {
		BaseConfig
		Timeout time.Duration
		Args    []string `delim:" "`
	}
	RegisterTaskFatal(TaskRepoHealthCheck, &RepoHealthCheckConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: false,
			Schedule:   scheduleMidnight,
		},
		Timeout: time.Duration(setting.Git.Timeout.Default) * time.Second,
		Args:    []string{},
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		rhcConfig := config.(*RepoHealthCheckConfig)
		// the git args are set by config, they can be safe to be trusted
		return repo_service.GitFsckRepos(ctx, rhcConfig.Timeout, git.ToTrustedCmdArgs(rhcConfig.Args))
	})
}

func registerCheckRepoStats() {
	RegisterTaskFatal(TaskCheckRepoStats, &BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   scheduleMidnight,
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		return models.CheckRepoStats(ctx)
	})
}

func registerArchiveCleanup() {
	RegisterTaskFatal(TaskArchiveCleanup, &OlderThanConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: true,
			Schedule:   scheduleMidnight,
		},
		OlderThan: 24 * time.Hour,
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		acConfig := config.(*OlderThanConfig)
		return archiver_service.DeleteOldRepositoryArchives(ctx, acConfig.OlderThan)
	})
}

func registerSyncExternalUsers() {
	RegisterTaskFatal(TaskSyncExternalUsers, &UpdateExistingConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: false,
			Schedule:   scheduleMidnight,
		},
		UpdateExisting: true,
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		realConfig := config.(*UpdateExistingConfig)
		return auth.SyncExternalUsers(ctx, realConfig.UpdateExisting)
	})
}

func registerDeletedBranchesCleanup() {
	RegisterTaskFatal(TaskDeletedBranchesCleanup, &OlderThanConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: true,
			Schedule:   scheduleMidnight,
		},
		OlderThan: 24 * time.Hour,
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		realConfig := config.(*OlderThanConfig)
		git_model.RemoveOldDeletedBranches(ctx, realConfig.OlderThan)
		return nil
	})
}

func registerUpdateMigrationPosterID() {
	RegisterTaskFatal(TaskUpdateMigrationPosterId, &BaseConfig{
		Enabled:    true,
		RunAtStart: true,
		Schedule:   scheduleMidnight,
	}, func(ctx context.Context, _ *user_model.User, _ Config) error {
		return migrations.UpdateMigrationPosterID(ctx)
	})
}

func registerCleanupHookTaskTable() {
	RegisterTaskFatal(TaskCleanupHookTaskTable, &CleanupHookTaskConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: false,
			Schedule:   scheduleMidnight,
		},
		CleanupType:  "OlderThan",
		OlderThan:    168 * time.Hour,
		NumberToKeep: 10,
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		realConfig := config.(*CleanupHookTaskConfig)
		return webhook.CleanupHookTaskTable(ctx, webhook.ToHookTaskCleanupType(realConfig.CleanupType), realConfig.OlderThan, realConfig.NumberToKeep)
	})
}

func registerCleanupPackages() {
	RegisterTaskFatal(TaskCleanupPackages, &OlderThanConfig{
		BaseConfig: BaseConfig{
			Enabled:    true,
			RunAtStart: true,
			Schedule:   scheduleMidnight,
		},
		OlderThan: 24 * time.Hour,
	}, func(ctx context.Context, _ *user_model.User, config Config) error {
		realConfig := config.(*OlderThanConfig)
		return packages_cleanup_service.CleanupTask(ctx, realConfig.OlderThan)
	})
}

func initBasicTasks() {
	if setting.Mirror.Enabled {
		registerUpdateMirrorTask()
	}
	registerRepoHealthCheck()
	registerCheckRepoStats()
	registerArchiveCleanup()
	registerSyncExternalUsers()
	registerDeletedBranchesCleanup()
	if !setting.Repository.DisableMigrations {
		registerUpdateMigrationPosterID()
	}
	registerCleanupHookTaskTable()
	if setting.Packages.Enabled {
		registerCleanupPackages()
	}
}
