// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package resource

import (
	"context"
	"errors"
	"fmt"

	"forgejo.org/models/db"
	"forgejo.org/modules/util"

	"xorm.io/builder"
)

type Kind int8

const (
	KindOwner      Kind = 1
	KindRepository Kind = 2
)

type Resource struct {
	ID         int64 `xorm:"pk autoincr"`
	MirrorID   int64 `xorm:"INDEX UNIQUE(f3_resource_index) REFERENCES(f3_mirror, id)"`
	ResourceID int64 `xorm:"INDEX UNIQUE(f3_resource_index)"`
	Kind       Kind  `xorm:"INDEX UNIQUE(f3_resource_index)"`
}

func NewResource(forge, resource int64, kind Kind) *Resource {
	return &Resource{
		MirrorID:   forge,
		ResourceID: resource,
		Kind:       kind,
	}
}

func (o *Resource) SetID(id int64) {
	o.ID = id
}

func (o Resource) GetID() int64 {
	return o.ID
}

func (o *Resource) SetMirrorID(mirrorID int64) {
	o.MirrorID = mirrorID
}

func (o Resource) GetMirrorID() int64 {
	return o.MirrorID
}

func (o *Resource) SetResourceID(resourceID int64) {
	o.ResourceID = resourceID
}

func (o *Resource) GetResourceID() int64 {
	return o.ResourceID
}

func (o Resource) GetKind() Kind {
	return o.Kind
}

func (o *Resource) SetKind(kind Kind) {
	o.Kind = kind
}

func (o *Resource) TableName() string {
	return "f3_resource"
}

func init() {
	db.RegisterModel(new(Resource))
}

func Equal(a, b *Resource) bool {
	return a.MirrorID == b.MirrorID &&
		a.ResourceID == b.ResourceID &&
		a.Kind == b.Kind
}

func Upsert(ctx context.Context, resource *Resource) (*Resource, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		opts := FindOptions{
			MirrorID:   &resource.MirrorID,
			ResourceID: &resource.ResourceID,
			Kind:       &resource.Kind,
		}
		found, err := Get(ctx, opts)
		if err != nil && !errors.Is(err, util.ErrNotExist) {
			return fmt.Errorf("resource Upsert Get(%v): %w", opts, err)
		}
		if found == nil {
			return Insert(ctx, resource)
		}
		resource.SetID(found.GetID())
		if !Equal(resource, found) {
			if err := Update(ctx, resource); err != nil {
				return fmt.Errorf("resource Update(%v): %w", resource.ID, err)
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("resource Upsert WithTx: %w", err)
	}
	return resource, nil
}

func Update(ctx context.Context, new *Resource) error {
	if _, err := db.GetEngine(ctx).ID(new.ID).AllCols().Update(new); err != nil {
		return fmt.Errorf("resource Update Update(%v): %w", new, err)
	}
	return nil
}

func Insert(ctx context.Context, resource *Resource) error {
	if err := db.Insert(ctx, resource); err != nil {
		return fmt.Errorf("resource Insert Insert(%v): %w", resource, err)
	}
	return nil
}

type FindOptions struct {
	db.ListOptions
	MirrorID   *int64
	ResourceID *int64
	Kind       *Kind
}

func (opts FindOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.MirrorID != nil {
		cond = cond.And(builder.Eq{"mirror_id": opts.MirrorID})
	}
	if opts.ResourceID != nil {
		cond = cond.And(builder.Eq{"resource_id": opts.ResourceID})
	}
	if opts.Kind != nil {
		cond = cond.And(builder.Eq{"kind": opts.Kind})
	}
	return cond
}

type ResourceList []*Resource //revive:disable-line:exported

func Find(ctx context.Context, opts FindOptions) (ResourceList, error) {
	resourceList, err := db.Find[Resource](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("resource Find Find(%v): %w", opts, err)
	}
	return resourceList, nil
}

func Get(ctx context.Context, opts FindOptions) (*Resource, error) {
	resources, err := db.Find[Resource](ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("resource Get Find(%v): %w", opts, err)
	}
	if len(resources) == 0 {
		return nil, util.ErrNotExist
	}
	if len(resources) != 1 {
		return nil, fmt.Errorf("resource Get(%v) expected to find one resource but found %d instead", opts, len(resources))
	}
	return resources[0], nil
}
