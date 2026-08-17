// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models/db"
	org_model "forgejo.org/models/organization"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

type teams struct {
	container
}

func (o *teams) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	pageSize := o.getPageSize()

	organizationID := f3_tree.GetOwnerID(node)

	forgejoTeams, _, err := org_model.SearchTeam(ctx, &org_model.SearchTeamOptions{
		ListOptions: db.ListOptions{Page: page, PageSize: pageSize},
		OrgID:       organizationID,
	})
	if err != nil {
		panic(fmt.Errorf("error while listing teams: %v", err))
	}

	for _, forgejoTeam := range forgejoTeams {
		if err := forgejoTeam.LoadUnits(ctx); err != nil {
			panic(fmt.Errorf("LoadUnits(%+v): %w", forgejoTeam, err))
		}
	}
	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(forgejoTeams...)...)
}

func (o *teams) LookupMappedID(ctx context.Context, id f3_id.NodeID, f f3.Interface) f3_id.NodeID {
	team := f.(*f3.Team)
	return o.GetIDFromName(ctx, team.Name)
}

func (o *teams) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	organizationID := f3_tree.GetOrganizationID(o.GetNode())
	opts := &org_model.SearchTeamOptions{
		OrgID:   organizationID,
		Keyword: name,
	}
	forgejoTeams, _, err := org_model.SearchTeam(ctx, opts)
	if err != nil {
		panic(fmt.Errorf("error SearchTeam(%v): %v", opts, err))
	}

	var teamID int64
	for _, forgejoTeam := range forgejoTeams {
		if forgejoTeam.Name == name {
			teamID = forgejoTeam.ID
			break
		}
	}

	if teamID == 0 {
		return f3_id.NilID
	}

	return f3_id.NewNodeID(f3_util.ToString(teamID))
}

func newTeams() f3_tree_generic.NodeDriverInterface {
	return &teams{}
}
