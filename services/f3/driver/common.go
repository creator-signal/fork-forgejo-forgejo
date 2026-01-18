// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	issues_model "forgejo.org/models/issues"
	org_model "forgejo.org/models/organization"
	f3_context "forgejo.org/services/f3/context"

	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_kind "code.forgejo.org/f3/gof3/v3/kind"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	f3_tree_generic "code.forgejo.org/f3/gof3/v3/tree/generic"
)

type common struct {
	f3_tree_generic.NullDriver
}

func (o *common) GetHelper() any {
	panic("not implemented")
}

func (o *common) ListPage(ctx context.Context, node f3_tree_generic.NodeInterface, _ f3_tree_generic.ListOptions, page int) f3_tree_generic.ChildrenList {
	return f3_tree_generic.NewChildrenList(0)
}

func (o *common) GetNativeID() string {
	return ""
}

func (o *common) SetNative(native any) {
}

func (o *common) getTree() f3_tree_generic.TreeInterface {
	return o.GetNode().GetTree()
}

func (o *common) getPageSize() int {
	return o.getTreeDriver().GetPageSize()
}

func (o *common) getForge() *forge {
	return o.getTree().GetRoot().GetChild(f3_id.NewNodeID(f3_kind.KindForge)).GetDriver().(*forge)
}

func (o *common) getForgejoForgeID(ctx context.Context) int64 {
	return o.getForge().getForgejoForgeID(ctx)
}

func (o *common) getKind() f3_kind.Kind {
	return o.GetNode().GetKind()
}

func (o *common) getTreeDriver() *treeDriver {
	return o.GetTreeDriver().(*treeDriver)
}

func (o *common) IsNull() bool { return false }

func (o *common) getIssueOrPullRequestAbsoluteID(ctx context.Context, node f3_tree_generic.NodeInterface) int64 {
	project := f3_tree.GetProjectID(node)
	issueOrPullRequest := f3_tree.GetIssueOrPullRequest(node)
	index := issueOrPullRequest.GetID().Int64()
	issue, err := issues_model.GetIssueByIndex(ctx, project, index)
	if err != nil {
		panic(fmt.Errorf("GetIssueByIndex %v %w", index, err))
	}
	return issue.ID
}

func (o *common) getTeam(ctx context.Context) *org_model.Team {
	node := o.GetNode()

	teamID := f3_tree.GetTeamID(node)

	team, err := org_model.GetTeamByID(ctx, teamID)
	if err != nil {
		panic(fmt.Errorf("GetTeamByID(%v): %w", teamID, err))
	}
	return team
}

func (o *common) sendNotifications(ctx context.Context) bool {
	f3Ctx := f3_context.Get(ctx)
	if f3Ctx == nil {
		return true
	}
	return f3Ctx.GetSendNotifications()
}
