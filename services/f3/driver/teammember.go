// Copyright Forgejo Authors
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"forgejo.org/models"
	"forgejo.org/models/db"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &teamMember{}

type forgejoTeamMember struct {
	ID     int64
	Name   string
	UserID int64
}

func getForgejoTeamMemberByID(ctx context.Context, id int64) *forgejoTeamMember {
	var member forgejoTeamMember
	has, err := db.GetEngine(ctx).
		Select("`team_user`.`id` as `id`, `user`.`name` as `name`, `user`.`id` as `user_id`").
		Table("`team_user`").
		Join("INNER", "`user`", "`user`.`id` = `team_user`.`uid`").
		Where("`team_user`.`id` = ?", id).
		Get(&member)
	if err != nil {
		panic(fmt.Errorf("looking for member %v: %w", id, err))
	}
	if !has {
		return nil
	}
	return &member
}

func getForgejoTeamMemberByTeamIDAndName(ctx context.Context, teamID int64, name string) *forgejoTeamMember {
	var member forgejoTeamMember
	has, err := db.GetEngine(ctx).
		Select("`team_user`.`id` as `id`, `user`.`name` as `name`, `user`.`id` as `user_id`").
		Table("user").
		Join("INNER", "`team_user`", "`team_user`.`uid` = `user`.`id`").
		Where("`user`.`name` = ?", name).
		And("`team_user`.`team_id` = ?", teamID).
		Get(&member)
	if err != nil {
		panic(fmt.Errorf("looking for member %v: %w", name, err))
	}
	if !has {
		return nil
	}
	return &member
}

func getForgejoTeamMembersByTeamID(ctx context.Context, teamID int64, pageSize, page int) []*forgejoTeamMember {
	var members []*forgejoTeamMember
	sess := db.GetEngine(ctx)
	if pageSize > 0 && page > 0 {
		sess = sess.Limit(pageSize, (page-1)*pageSize)
	}
	if err := sess.Select("`team_user`.`id` as `id`, `user`.`id` as `user_id`, `user`.`name` as `name`").
		Table("`team_user`").
		Join("INNER", "`team`", "`team`.`id` = `team_user`.`team_id`").
		Join("INNER", "`user`", "`user`.`id` = `team_user`.`uid`").
		Where("`team`.`id` = ?", teamID).
		Find(&members); err != nil {
		panic(fmt.Errorf("teamID=%v pageSize=%v page=%v", teamID, pageSize, page))
	}
	return members
}

type teamMember struct {
	common

	member *forgejoTeamMember
}

func (o *teamMember) SetNative(member any) {
	o.member = member.(*forgejoTeamMember)
}

func (o *teamMember) GetNativeID() string {
	return fmt.Sprintf("%d", o.member.ID)
}

func (o *teamMember) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *teamMember) ToFormat() f3.Interface {
	if o.member == nil {
		return o.NewFormat()
	}

	return (&f3.TeamMember{
		Common: f3.NewCommon(o.GetNativeID()),
		User:   f3_tree.NewUserReference(f3_util.ToString(o.member.UserID)),
		Name:   o.member.Name,
	}).Init()
}

func (o *teamMember) FromFormat(content f3.Interface) {
	teamMember := content.(*f3.TeamMember)
	o.member = &forgejoTeamMember{
		ID:     f3_util.ParseInt(teamMember.GetID()),
		UserID: teamMember.User.GetIDAsInt(),
		Name:   teamMember.Name,
	}
}

func (o *teamMember) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	id := node.GetID().Int64()

	member := getForgejoTeamMemberByID(ctx, id)
	if member == nil {
		return false
	}
	o.member = member
	return true
}

func (o *teamMember) Patch(ctx context.Context) {
	panic("not implemented")
}

func (o *teamMember) Put(ctx context.Context) f3_id.NodeID {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	teamUser, err := models.InsertTeamMember(ctx, o.getTeam(ctx), o.member.UserID)
	if err != nil {
		panic(fmt.Errorf("AddAndReturnTeamMember(%+v): %w", o.member, err))
	}

	member := getForgejoTeamMemberByID(ctx, teamUser.ID)
	if member == nil {
		panic(fmt.Errorf("unable to retrieve %+v", teamUser))
	}
	o.member = member
	o.Trace("teamMember created %+v", o.member)
	return f3_id.NewNodeID(o.member.ID)
}

func (o *teamMember) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if err := models.RemoveTeamMember(ctx, o.getTeam(ctx), o.member.UserID); err != nil {
		panic(fmt.Errorf("RemoveTeamMember(%+v): %w", o.member, err))
	}
}

func newTeamMember() generic.NodeDriverInterface {
	return &teamMember{}
}
