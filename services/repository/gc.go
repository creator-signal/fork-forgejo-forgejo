// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"forgejo.org/models/db"
	repo_model "forgejo.org/models/repo"
	"forgejo.org/modules/git"
	"forgejo.org/modules/graceful"
	"forgejo.org/modules/log"
	"forgejo.org/modules/queue"
	repo_module "forgejo.org/modules/repository"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
)

var ErrGCCooldown = errors.New("garbage collection ran too recently")

type RepoGCOptions struct {
	RepoID int64
}

var repoGCQueue *queue.WorkerPoolQueue[*RepoGCOptions]

func handlerRepoGC(items ...*RepoGCOptions) []*RepoGCOptions {
	for _, opts := range items {
		repo, err := repo_model.GetRepositoryByID(graceful.GetManager().ShutdownContext(), opts.RepoID)
		if err != nil {
			log.Error("GC: GetRepositoryByID [%d] failed: %v", opts.RepoID, err)
			continue
		}
		if err := runGCForRepo(graceful.GetManager().ShutdownContext(), repo); err != nil {
			log.Error("GC: runGCForRepo [%d] failed: %v", opts.RepoID, err)
			continue
		}
		repo.LastGCUnix = timeutil.TimeStampNow()
		if _, err := db.GetEngine(graceful.GetManager().ShutdownContext()).ID(repo.ID).Cols("last_gc_unix").NoAutoTime().Update(repo); err != nil {
			log.Error("GC: update last_gc_unix [%d] failed: %v", opts.RepoID, err)
		}
	}
	return nil
}

func initRepoGCQueue(ctx context.Context) error {
	repoGCQueue = queue.CreateUniqueQueue(ctx, "repo_gc", handlerRepoGC)
	if repoGCQueue == nil {
		return errors.New("unable to create repo_gc queue")
	}
	go graceful.GetManager().RunWithCancel(repoGCQueue)
	return nil
}

func EnqueueRepoGC(ctx context.Context, repo *repo_model.Repository) error {
	if setting.Repository.GCCooldownMinutes > 0 {
		cooldown := time.Duration(setting.Repository.GCCooldownMinutes) * time.Minute
		if repo.LastGCUnix > 0 && time.Since(repo.LastGCUnix.AsTime()) < cooldown {
			return ErrGCCooldown
		}
	}
	return repoGCQueue.Push(&RepoGCOptions{RepoID: repo.ID})
}

func runGCForRepo(ctx context.Context, repo *repo_model.Repository) error {
	reflogCmd := git.NewCommand(ctx, "reflog", "expire", "--expire-unreachable=now", "--all").
		SetDescription(fmt.Sprintf("Reflog expire: %s", repo.FullName()))
	if _, _, err := reflogCmd.RunStdString(&git.RunOpts{
		Timeout: time.Duration(setting.Git.Timeout.GC) * time.Second,
		Dir:     repo.RepoPath(),
	}); err != nil {
		log.Warn("Reflog expire failed for %-v: %v", repo, err)
	}

	maintenanceCmd := git.NewCommand(ctx, "-c", "gc.pruneExpire=now", "maintenance", "run", "--task=gc").
		SetDescription(fmt.Sprintf("Git maintenance: %s", repo.FullName()))
	if _, _, err := maintenanceCmd.RunStdString(&git.RunOpts{
		Timeout: time.Duration(setting.Git.Timeout.GC) * time.Second,
		Dir:     repo.RepoPath(),
	}); err != nil {
		return err
	}

	if err := repo_module.UpdateRepoSize(ctx, repo); err != nil {
		log.Error("UpdateRepoSize [%d] failed: %v", repo.ID, err)
	}

	return GarbageCollectLFSMetaObjectsForRepo(ctx, repo, GarbageCollectLFSMetaObjectsOptions{
		LogDetail: log.Trace,
		AutoFix:   true,
	})
}
