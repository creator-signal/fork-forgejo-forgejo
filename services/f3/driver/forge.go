// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	f3_forge_model "forgejo.org/models/f3/forge"
	user_model "forgejo.org/models/user"
	f3_mirror_service "forgejo.org/services/f3/mirror"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	"code.forgejo.org/f3/gof3/v3/util"
)

type forge struct {
	generic.NullDriver

	ownersKind map[string]f3_kind.Kind
	forge      *f3_forge_model.Forge
	url        string
}

func newForge() generic.NodeDriverInterface {
	return &forge{
		ownersKind: make(map[string]f3_kind.Kind),
	}
}

func (o *forge) getOwnersKind(ctx context.Context, id string) f3_kind.Kind {
	kind, ok := o.ownersKind[id]
	if !ok {
		user, err := user_model.GetUserByID(ctx, util.ParseInt(id))
		if err != nil {
			panic(fmt.Errorf("user_repo.GetUserByID: %w", err))
		}
		kind = f3_kind.KindUsers
		if user.IsOrganization() {
			kind = f3_kind.KindOrganizations
		}
		o.ownersKind[id] = kind
	}
	return kind
}

func (o *forge) getForgejoForge(ctx context.Context) *f3_forge_model.Forge {
	if o.forge == nil {
		opts := f3_forge_model.FindOptions{
			URL: &o.url,
		}
		forge, err := f3_forge_model.Get(ctx, opts)
		if err != nil {
			panic(err)
		}
		o.forge = forge
	}
	return o.forge
}

func (o *forge) getForgejoForgeID(ctx context.Context) int64 {
	return o.getForgejoForge(ctx).ID
}

func (o *forge) getOwnersPath(ctx context.Context, id string) generic.Path {
	return generic.NewPathFromString("/").SetForge().SetOwners(o.getOwnersKind(ctx, id))
}

func (o *forge) Get(ctx context.Context) bool {
	return o.getForgejoForge(ctx) != nil
}

func (o *forge) Put(ctx context.Context) f3_id.NodeID {
	forge, err := f3_mirror_service.UpsertForge(ctx, o.url, "", "")
	if err != nil {
		panic(fmt.Errorf("UpsertForge %s: %w", o.url, err))
	}
	o.forge = forge
	return f3_id.NewNodeID("forge")
}

func (o *forge) Patch(context.Context)   {}
func (o *forge) Delete(context.Context)  {}
func (o *forge) NewFormat() f3.Interface { return (&f3.Forge{}).Init() }
func (o *forge) FromFormat(content f3.Interface) {
	forge := content.(*f3.Forge)
	o.url = forge.URL
}

func (o *forge) ToFormat() f3.Interface {
	return (&f3.Forge{
		Common: f3.NewCommon("forge"),
		URL:    o.url,
	}).Init()
}
