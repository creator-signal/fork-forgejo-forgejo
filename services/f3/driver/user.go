// Copyright Earl Warren <contact@earl-warren.org>
// Copyright Loïc Dachary <loic@dachary.org>
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"
	"strings"

	f3_resource_model "forgejo.org/models/f3/resource"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/optional"
	permissions_user "forgejo.org/services/permissions/user"
	user_service "forgejo.org/services/user"

	"code.forgejo.org/f3/gof3/v3/f3"
	f3_id "code.forgejo.org/f3/gof3/v3/id"
	f3_tree "code.forgejo.org/f3/gof3/v3/tree/f3"
	"code.forgejo.org/f3/gof3/v3/tree/generic"
	f3_util "code.forgejo.org/f3/gof3/v3/util"
)

var _ f3_tree.ForgeDriverInterface = &user{}

type user struct {
	common

	forgejoUser *user_model.User
	avatar      string
}

const fakeEmailSuffix = ".fakeemail"

func fromFakeEmail(mail string) string {
	return strings.TrimSuffix(mail, fakeEmailSuffix)
}

func toFakeEmail(mail string) string {
	if !strings.HasSuffix(mail, fakeEmailSuffix) {
		return mail + fakeEmailSuffix
	}
	return mail
}

func getSystemUserByName(name string) *user_model.User {
	switch name {
	case user_model.GhostUserName:
		return user_model.NewGhostUser()
	case user_model.ActionsUserName:
		return user_model.NewActionsUser()
	default:
		return nil
	}
}

func (o *user) SetNative(user any) {
	o.forgejoUser = user.(*user_model.User)
}

func (o *user) GetNativeID() string {
	return fmt.Sprintf("%d", o.forgejoUser.ID)
}

func (o *user) NewFormat() f3.Interface {
	node := o.GetNode()
	return node.GetTree().(f3_tree.TreeInterface).NewFormat(node.GetKind())
}

func (o *user) ToFormat() f3.Interface {
	if o.forgejoUser == nil {
		return o.NewFormat()
	}
	return (&f3.User{
		Common:   f3.NewCommon(fmt.Sprintf("%d", o.forgejoUser.ID)),
		UserName: o.forgejoUser.Name,
		Name:     o.forgejoUser.FullName,
		Email:    o.forgejoUser.Email,
		IsAdmin:  o.forgejoUser.IsAdmin,
		Avatar:   o.avatar,
		Password: o.forgejoUser.Passwd,
	}).Init()
}

func (o *user) FromFormat(content f3.Interface) {
	user := content.(*f3.User)
	o.forgejoUser = &user_model.User{
		Type:     user_model.UserTypeRemoteUser,
		ID:       f3_util.ParseInt(user.GetID()),
		Name:     user.UserName,
		FullName: user.Name,
		Email:    user.Email,
		IsAdmin:  user.IsAdmin,
		Passwd:   user.Password,
	}
	o.avatar = user.Avatar
}

func (o *user) GetFormatCompareInfo(f any, field string) f3.CompareInfo {
	switch field {
	case "Email":
		return f3.NewCompareInfo().SetTransform(func(v any) any {
			return fromFakeEmail(v.(string))
		})
	default:
		return o.common.GetFormatCompareInfo(f, field)
	}
}

func (o *user) Get(ctx context.Context) bool {
	node := o.GetNode()
	o.Trace("%s", node.GetID())
	id := node.GetID().Int64()
	u, err := user_model.GetPossibleUserByID(ctx, id)
	if user_model.IsErrUserNotExist(err) {
		return false
	}
	if err != nil {
		panic(fmt.Errorf("user %v %w", id, err))
	}
	u.Email = fromFakeEmail(u.Email)
	o.forgejoUser = u
	o.avatar = o.getUserAvatar(ctx, o.forgejoUser)
	return true
}

func (o *user) Patch(ctx context.Context) {
	o.setUserAvatar(ctx, o.forgejoUser, o.avatar)
}

func (o *user) Put(ctx context.Context) f3_id.NodeID {
	if user := getSystemUserByName(o.forgejoUser.Name); user != nil {
		return f3_id.NewNodeID(user.ID)
	}
	permissionsCheck(ctx, permissions_user.Put(o.forgejoUser))

	o.forgejoUser.LowerName = strings.ToLower(o.forgejoUser.Name)
	o.forgejoUser.Email = toFakeEmail(o.forgejoUser.Email)
	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive: optional.Some(true),
	}
	if err := user_model.CreateUser(ctx, o.forgejoUser, overwriteDefault); err != nil {
		panic(err)
	}
	o.Trace("user created %s/%d", o.forgejoUser.Name, o.forgejoUser.ID)
	_, err := f3_resource_model.Upsert(ctx, f3_resource_model.NewResource(o.getForgejoForgeID(ctx), o.forgejoUser.ID, f3_resource_model.KindOwner))
	if err != nil {
		panic(err)
	}
	o.setUserAvatar(ctx, o.forgejoUser, o.avatar)
	return f3_id.NewNodeID(o.forgejoUser.ID)
}

func (o *user) Delete(ctx context.Context) {
	node := o.GetNode()
	o.Trace("%s", node.GetID())

	if o.forgejoUser.ID == 1 && o.forgejoUser.IsAdmin {
		o.Debug("silently ignore a request to delete the admin user with ID 1 because it is assumed to be required: %+v", o.forgejoUser.Type)
		return
	}

	if err := user_service.DeleteUser(ctx, o.forgejoUser, true); err != nil {
		panic(err)
	}
}

func newUser() generic.NodeDriverInterface {
	return &user{}
}
