// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package resource

import (
	"context"
	"fmt"

	"forgejo.org/models/db"

	"xorm.io/builder"
)

type Kind int8

const (
	KindOwner      Kind = 1
	KindRepository Kind = 2
)

type Resource struct {
	ID         int64 `xorm:"pk autoincr"`
	ForgeID    int64 `xorm:"INDEX UNIQUE(f3_resource_index) REFERENCES(f3_forge, id)"`
	ResourceID int64 `xorm:"INDEX UNIQUE(f3_resource_index)"`
	Kind       Kind  `xorm:"INDEX UNIQUE(f3_resource_index)"`
}

func NewResource(forge, resource int64, kind Kind) *Resource {
	return &Resource{
		ForgeID:    forge,
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

func (o *Resource) SetForgeID(forgeID int64) {
	o.ForgeID = forgeID
}

func (o Resource) GetForgeID() int64 {
	return o.ForgeID
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
	return a.ForgeID == b.ForgeID &&
		a.ResourceID == b.ResourceID &&
		a.Kind == b.Kind
}

func Upsert(ctx context.Context, resource *Resource) (*Resource, error) {
	if err := db.WithTx(ctx, func(ctx context.Context) error {
		found, err := Get(ctx, FindOptions{
			ForgeID:    &resource.ForgeID,
			ResourceID: &resource.ResourceID,
			Kind:       &resource.Kind,
		})
		if err != nil {
			return err
		}
		if found == nil {
			return Insert(ctx, resource)
		}
		resource.SetID(found.GetID())
		if !Equal(resource, found) {
			if err := Update(ctx, resource); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return resource, nil
}

func Update(ctx context.Context, new *Resource) error {
	_, err := db.GetEngine(ctx).ID(new.ID).Update(new)
	return err
}

func Insert(ctx context.Context, resource *Resource) error {
	return db.Insert(ctx, resource)
}

type FindOptions struct {
	db.ListOptions
	ForgeID    *int64
	ResourceID *int64
	Kind       *Kind
}

func (opts FindOptions) ToConds() builder.Cond {
	cond := builder.NewCond()
	if opts.ForgeID != nil {
		cond = cond.And(builder.Eq{"forge_id": opts.ForgeID})
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
	return db.Find[Resource](ctx, opts)
}

func Get(ctx context.Context, opts FindOptions) (*Resource, error) {
	resources, err := db.Find[Resource](ctx, opts)
	if err != nil {
		return nil, err
	}
	if len(resources) == 0 {
		return nil, nil //nolint:nilnil
	}
	if len(resources) != 1 {
		return nil, fmt.Errorf("expected to find one resource but found %d instead", len(resources))
	}
	return resources[0], nil
}
