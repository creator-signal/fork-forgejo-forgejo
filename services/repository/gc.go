// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT
package repository

import (
	"context"
	"fmt"
	"time"

	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

func RunManualGCForRepo(ctx context.Context, repo *repo_model.Repository) error {
	cmd := git.NewCommand(ctx, "reflog", "expire", "--expire-unreachable=now", "--all").
		SetDescription(fmt.Sprintf("Reflog expire: %s", repo.FullName()))
	if _, _, err := cmd.RunStdString(&git.RunOpts{
		Timeout: time.Duration(setting.Git.Timeout.GC) * time.Second,
		Dir:     repo.RepoPath(),
	}); err != nil {
		log.Warn("Reflog expire failed for %-v: %v", repo, err)
	}

	if err := GitGcRepo(
		ctx, repo,
		time.Duration(setting.Git.Timeout.GC)*time.Second,
		git.ToTrustedCmdArgs(setting.Git.GCArgs),
	); err != nil {
		return err
	}

	return GarbageCollectLFSMetaObjectsForRepo(ctx, repo, GarbageCollectLFSMetaObjectsOptions{
		LogDetail: log.Trace,
		AutoFix:   true,
	})
}
