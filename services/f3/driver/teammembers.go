// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver //nolint:dupl

import (
	"context"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

type teamMembers struct {
	container
}

func (o *teamMembers) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	pageSize := o.getPageSize()

	teamID := f3_tree.GetTeamID(node)

	forgejoTeamMembers := getForgejoTeamMembersByTeamID(ctx, teamID, pageSize, page)

	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(forgejoTeamMembers...)...)
}

func (o *teamMembers) LookupMappedID(ctx context.Context, id f3_id.NodeID, f f3.Interface) f3_id.NodeID {
	teamMember := f.(*f3.TeamMember)
	i := o.GetIDFromName(ctx, teamMember.Name)
	o.Trace("%v => %v", teamMember, i)
	return i
}

func (o *teamMembers) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	teamID := f3_tree.GetTeamID(o.GetNode())
	member := getForgejoTeamMemberByTeamIDAndName(ctx, teamID, name)
	if member == nil {
		return f3_id.NilID
	}

	return f3_id.NewNodeID(f3_util.ToString(member.ID))
}

func newTeamMembers() f3_tree_generic.NodeDriverInterface {
	return &teamMembers{}
}
