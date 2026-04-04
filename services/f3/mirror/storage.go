// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package mirror

import (
	"context"

	"forgejo.org/models/db"
	f3_forge_model "forgejo.org/models/f3/forge"
	f3_mirror_model "forgejo.org/models/f3/mirror"
	f3_resource_model "forgejo.org/models/f3/resource"
)

func GetMirror(ctx context.Context, id int64) (*f3_mirror_model.Mirror, error) {
	mirror, err := f3_mirror_model.Get(ctx, f3_mirror_model.FindOptions{
		ID: &id,
	})
	if err != nil {
		return nil, err
	}
	if err := mirror.LoadForge(ctx); err != nil {
		return nil, err
	}
	return mirror, nil
}

func UpsertForge(ctx context.Context, url, typ string) (*f3_forge_model.Forge, error) {
	forge := f3_forge_model.NewForge()
	forge.SetURL(url)
	forge.SetType(typ)
	return f3_forge_model.Upsert(ctx, forge)
}

func DeleteForge(ctx context.Context, url string) error {
	forge, err := f3_forge_model.Get(ctx, f3_forge_model.FindOptions{URL: &url})
	if err != nil || forge == nil {
		return err
	}
	return db.DeleteBeans(ctx,
		&f3_forge_model.Forge{ID: forge.ID},
		&f3_resource_model.Resource{MirrorID: forge.ID},
		&f3_mirror_model.Mirror{ForgeID: forge.ID},
	)
}
