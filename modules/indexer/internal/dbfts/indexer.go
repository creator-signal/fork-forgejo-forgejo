// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package dbfts

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/indexer/internal"
)

var _ internal.Indexer = &Indexer{}

// Indexer represents a db indexer using fts extensions if available
type Indexer struct {
	// TODO
	// version int
	name   string
	create Func
}

type Func func(context.Context) error

func NewIndexer(name string, create Func) *Indexer {
	return &Indexer{
		name:   name,
		create: create,
	}
}

func (i *Indexer) Init(ctx context.Context) (bool, error) {
	// TODO: replace with i.has()
	has, err := db.GetEngine(ctx).IsTableExist(i.name)
	if err != nil {
		return has, err
	}

	if !has {
		err = i.create(ctx)
	}
	return has, err
}

// Dummy Functions
func (i *Indexer) Ping(_ context.Context) error { return nil }
func (i *Indexer) Close()                       {}
