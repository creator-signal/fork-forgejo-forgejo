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

type teamProjects struct {
	container
}

func (o *teamProjects) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	pageSize := o.getPageSize()

	teamID := f3_tree.GetTeamID(node)

	forgejoTeamProjects := getForgejoTeamReposByTeamID(ctx, teamID, pageSize, page)

	return f3_tree.ConvertListed(ctx, node, f3_tree.ConvertToAny(forgejoTeamProjects...)...)
}

func (o *teamProjects) LookupMappedID(ctx context.Context, id f3_id.NodeID, f f3.Interface) f3_id.NodeID {
	teamProject := f.(*f3.TeamProject)
	i := o.GetIDFromName(ctx, teamProject.Name)
	o.Trace("%v => %v", teamProject, i)
	return i
}

func (o *teamProjects) GetIDFromName(ctx context.Context, name string) f3_id.NodeID {
	teamID := f3_tree.GetTeamID(o.GetNode())
	project := getForgejoTeamRepoByTeamIDAndName(ctx, teamID, name)
	if project == nil {
		return f3_id.NilID
	}

	return f3_id.NewNodeID(f3_util.ToString(project.ID))
}

func newTeamProjects() f3_tree_generic.NodeDriverInterface {
	return &teamProjects{}
}
