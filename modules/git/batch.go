// Copyright 2024 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package git

import (
	"bufio"
	"context"
	"io"
)

type Batch struct {
	cancel context.CancelFunc
	Reader *bufio.Reader
	Writer io.Writer
}

func (repo *Repository) newBatch(ctx context.Context) (*Batch, error) {
	// Now because of some insanity with git cat-file not immediately failing if not run in a valid git directory we need to run git rev-parse first!
	if err := ensureValidGitRepository(ctx, repo.Path); err != nil {
		return nil, err
	}

	var batch Batch
	var err error
	batch.Writer, batch.Reader, batch.cancel, err = catFileBatch(ctx, repo.Path)
	return &batch, err
}

func (repo *Repository) newBatchCheck(ctx context.Context) (*Batch, error) {
	// Now because of some insanity with git cat-file not immediately failing if not run in a valid git directory we need to run git rev-parse first!
	if err := ensureValidGitRepository(ctx, repo.Path); err != nil {
		return nil, err
	}

	var check Batch
	var err error
	check.Writer, check.Reader, check.cancel, err = catFileBatchCheck(ctx, repo.Path)
	return &check, err
}

func (b *Batch) Close() {
	if b.cancel != nil {
		b.cancel()
		b.Reader = nil
		b.Writer = nil
		b.cancel = nil
	}
}
