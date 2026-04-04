// Copyright The Forgejo Authors.
// SPDX-License-Identifier: GPLv3-or-later

package mirror

import (
	"context"

	"forgejo.org/models/db"
	f3_forge_model "forgejo.org/models/f3/forge"
	f3_mirror_model "forgejo.org/models/f3/mirror"
	f3_resource_model "forgejo.org/models/f3/resource"
)

func GetForge(ctx context.Context, id int64) (*f3_forge_model.Forge, error) {
	return f3_forge_model.Get(ctx, f3_forge_model.FindOptions{
		ID: &id,
	})
}

func GetMirror(ctx context.Context, id int64) (*f3_mirror_model.Mirror, error) {
	return f3_mirror_model.Get(ctx, f3_mirror_model.FindOptions{
		ID: &id,
	})
}

func UpsertForge(ctx context.Context, url, token, typ string) (*f3_forge_model.Forge, error) {
	forge := f3_forge_model.NewForge()
	forge.SetURL(url)
	forge.SetToken(token)
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
		&f3_resource_model.Resource{ForgeID: forge.ID},
		&f3_mirror_model.Mirror{ForgeID: forge.ID},
	)
}
