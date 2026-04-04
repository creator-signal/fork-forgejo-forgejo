// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package forge

import (
	"context"
	"errors"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

type Forge struct { //revive:disable-line:exported
	ID   int64  `xorm:"pk autoincr"`
	URL  string `xorm:"INDEX UNIQUE"`
	Type string
}

func NewForge() *Forge {
	return &Forge{}
}

func (o *Forge) SetID(id int64) {
	o.ID = id
}

func (o Forge) GetID() int64 {
	return o.ID
}

func (o *Forge) SetURL(url string) {
	o.URL = url
}

func (o Forge) GetURL() string {
	return o.URL
}

func (o *Forge) SetType(t string) {
	o.Type = t
}

func (o Forge) GetType() string {
	return o.Type
}

func (o *Forge) TableName() string {
	return "f3_forge"
}

func init() {
	db.RegisterModel(new(Forge))
}

func Equal(a, b *Forge) bool {
	return true // immutable
}

func Upsert(ctx context.Context, forge *Forge) (*Forge, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		found, err := Get(ctx, FindOptions{URL: &forge.URL})
		if err != nil && !errors.Is(err, util.ErrNotExist) {
			return fmt.Errorf("forge Upsert Get(%v): %w", forge.URL, err)
		}
		if found == nil {
			return Insert(ctx, forge)
		}
		forge.SetID(found.GetID())
		if !Equal(forge, found) {
			if err := Update(ctx, forge); err != nil {
				return fmt.Errorf("forge Update(%v): %w", forge.ID, err)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("forge Upsert WithTx: %w", err)
	}
	return forge, nil
}

func Update(ctx context.Context, new *Forge) error {
	if _, err := db.GetEngine(ctx).ID(new.ID).AllCols().Update(new); err != nil {
		return fmt.Errorf("forge Update Update(%v): %w", new.ID, err)
	}
	return nil
}

func Insert(ctx context.Context, forge *Forge) error {
	if err := db.Insert(ctx, forge); err != nil {
		return fmt.Errorf("forge Insert Insert: %w", err)
	}
	return nil
}

type FindOptions struct {
	db.ListOptions
	ID  *int64
	URL *string
}

func (opts FindOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.ID != nil {
		cond = cond.And(builder.Eq{"id": opts.ID})
	}
	if opts.URL != nil {
		cond = cond.And(builder.Eq{"url": opts.URL})
	}
	return cond
}

type ForgeList []*Forge //revive:disable-line:exported

func Find(ctx context.Context, opts FindOptions) (ForgeList, error) {
	forges, err := db.Find[Forge](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("forge Find Find: %w", err)
	}
	return forges, nil
}

func Get(ctx context.Context, opts FindOptions) (*Forge, error) {
	forges, err := db.Find[Forge](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("forge Get Find(%+v): %w", opts, err)
	}
	if len(forges) == 0 {
		return nil, util.ErrNotExist
	}
	if len(forges) != 1 {
		return nil, fmt.Errorf("forge Get(%+v) expected to find one forge but found %d instead", opts, len(forges))
	}
	return forges[0], nil
}
