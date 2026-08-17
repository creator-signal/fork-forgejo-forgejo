// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models"
	org_model "forgejo.org/models/organization"
	perm_model "forgejo.org/models/perm"
	unit_model "forgejo.org/models/unit"

	"code.forgejo.org/f3/gof3/v3/f3"
	"code.forgejo.org/f3/gof3/v3/f3/markdown"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
	f3_maps "code.forgejo.org/f3/gof3/v3/util/maps"
)

var _ f3_tree.ForgeDriverInterface = &team{}

type team struct {
	common

	forgejoTeam *org_model.Team
}

func (o *team) SetNative(team any) {
	o.forgejoTeam = team.(*org_model.Team)
}

func (o *team) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoTeam.ID)
}

func (o *team) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

var forgejoUnitToF3Unit = map[unit_model.Type]string{
	unit_model.TypeCode:         f3.TeamPermissionRepository,
	unit_model.TypeIssues:       f3.TeamPermissionIssue,
	unit_model.TypePullRequests: f3.TeamPermissionPullRequest,
	unit_model.TypeReleases:     f3.TeamPermissionRelease,
}

var f3UnitToForgejoUnit = f3_maps.Invert(forgejoUnitToF3Unit)

var forgejoAccessToF3Access = map[perm_model.AccessMode]string{
	perm_model.AccessModeNone:  f3.TeamAccessNone,
	perm_model.AccessModeRead:  f3.TeamAccessRead,
	perm_model.AccessModeWrite: f3.TeamAccessWrite,
	perm_model.AccessModeAdmin: f3.TeamAccessWrite,
	perm_model.AccessModeOwner: f3.TeamAccessWrite,
}

var f3AccessToForgejoAccess = map[string]perm_model.AccessMode{
	f3.TeamAccessNone:  perm_model.AccessModeNone,
	f3.TeamAccessRead:  perm_model.AccessModeRead,
	f3.TeamAccessWrite: perm_model.AccessModeWrite,
}

func (o *team) ToFormat() f3.Interface {
	if o.forgejoTeam == nil {
		return o.NewFormat()
	}
	var permissions []f3.TeamPermission

	switch o.forgejoTeam.AccessMode {
	case perm_model.AccessModeAdmin, perm_model.AccessModeOwner:
		permissions = []f3.TeamPermission{f3.TeamPermission(f3.TeamPermissionAdmin)}
	default:
		for _, forgejoUnit := range o.forgejoTeam.Units {
			unit, ok := forgejoUnitToF3Unit[forgejoUnit.Type]
			if !ok {
				o.Debug("repo unit %v %+v not supported and ignored", unit, forgejoUnit)
				continue
			}
			access, ok := forgejoAccessToF3Access[forgejoUnit.AccessMode]
			if !ok {
				panic(fmt.Errorf("unexpected access mode %v", forgejoUnit.AccessMode))
			}
			permissions = append(permissions, f3.TeamPermissionCompose(unit, access))
		}
	}

	return (&f3.Team{
		Common:      f3.NewCommon(o.GetNativeID()),
		Name:        o.forgejoTeam.Name,
		Description: markdown.NewContent().Set(o.forgejoTeam.Description),
		Permissions: permissions,
		AllProjects: o.forgejoTeam.IncludesAllRepositories,
	}).Init()
}

func (o *team) FromFormat(content f3.Interface) {
	team := content.(*f3.Team)

	var forgejoPermission perm_model.AccessMode
	var units []*org_model.TeamUnit

	if team.IsAdmin() {
		forgejoPermission = perm_model.AccessModeAdmin
	} else {
		forgejoPermission = perm_model.AccessModeWrite
		for _, permission := range team.Permissions {
			unit, access := f3.TeamPermissionSplit(permission)
			forgejoUnit, ok := f3UnitToForgejoUnit[unit]
			if !ok {
				panic(fmt.Errorf("unexpected unit %s in %v", unit, permission))
			}
			forgejoAccess, ok := f3AccessToForgejoAccess[access]
			if !ok {
				panic(fmt.Errorf("unexpected access %s in %v", access, permission))
			}
			units = append(units, &org_model.TeamUnit{
				Type:       forgejoUnit,
				AccessMode: forgejoAccess,
			})
		}
	}

	o.forgejoTeam = &org_model.Team{
		ID:                      f3_util.ParseInt(team.GetID()),
		Name:                    team.Name,
		Description:             team.Description.Get(),
		AccessMode:              forgejoPermission,
		Units:                   units,
		IncludesAllRepositories: team.AllProjects,
	}
}

func (o *team) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	team, err := org_model.GetTeamByID(ctx, id)
	if org_model.IsErrTeamNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("GetTeamByID(%v): %w", id, err))
	}

	if err := team.LoadUnits(ctx); err != nil {
		panic(fmt.Errorf("LoadUnits(%d): %w", id, err))
	}

	o.forgejoTeam = team
	return true
}

func (o *team) updateUnits(context.Context) {
	organizationID := f3_tree.GetOrganizationID(o.GetNode())
	for _, forgejoUnit := range o.forgejoTeam.Units {
		forgejoUnit.OrgID = organizationID
		forgejoUnit.TeamID = o.forgejoTeam.ID
	}
}

func (o *team) Patch(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	o.forgejoTeam.OrgID = f3_tree.GetOrganizationID(node)
	o.updateUnits(ctx)

	if err := models.UpdateTeam(ctx, o.forgejoTeam, true, true); err != nil {
		panic(fmt.Errorf("UpdateTeam(%+v): %w", o.forgejoTeam, err))
	}
}

func (o *team) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()

	o.forgejoTeam.OrgID = f3_tree.GetOrganizationID(node)
	o.updateUnits(ctx)

	if err := models.NewTeam(ctx, o.forgejoTeam); err != nil {
		panic(fmt.Errorf("NewTeam(%+v): %w", o.forgejoTeam, err))
	}

	o.Trace("team created %d", o.forgejoTeam.ID)
	return f3_id.NewNodeID(o.forgejoTeam.ID)
}

func (o *team) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := models.DeleteTeam(ctx, o.forgejoTeam); err != nil {
		panic(fmt.Errorf("DeleteTeam(%v): %w", o.forgejoTeam.ID, err))
	}
}

func newTeam() generic.NodeDriverInterface {
	return &team{}
}
