// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package context

import (
	"context"

	auth_model "forgejo.org/models/auth"
	access_model "forgejo.org/models/perm/access"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/services/authz"
	permissions_context "forgejo.org/services/permissions/context"
)

type key int

type F3 struct {
	Ctx               context.Context
	SendNotifications bool
	Permissions
}

func (o *F3) GetContext() context.Context {
	return o.Ctx
}

func (o *F3) SetSendNotifications(sendNotifications bool) {
	o.SendNotifications = sendNotifications
}

func (o *F3) GetSendNotifications() bool {
	return o.SendNotifications
}

var _ permissions_context.PermissionsContext = &F3{}

const (
	f3Key key = iota + 1
)

func WithF3(ctx context.Context) context.Context {
	return context.WithValue(ctx, f3Key, &F3{
		Ctx: ctx,
	})
}

type Permissions struct {
	Token      *auth_model.AccessToken
	Doer       *user_model.User
	Reducer    authz.AuthorizationReducer
	PublicOnly bool
	Repository *repo_model.Repository
	Permission *access_model.Permission
}

func Get(ctx context.Context) *F3 {
	if value := ctx.Value(f3Key); value != nil {
		return value.(*F3)
	}
	return nil
}

func (o *Permissions) GetRepository() *repo_model.Repository {
	return o.Repository
}

func (o *Permissions) SetRepository(repository *repo_model.Repository) {
	o.Repository = repository
}

func (o *Permissions) GetPermission() *access_model.Permission {
	return o.Permission
}

func (o *Permissions) SetPermission(permission *access_model.Permission) {
	o.Permission = permission
}

func (o *Permissions) GetToken() *auth_model.AccessToken {
	return o.Token
}

func (o *Permissions) SetToken(token *auth_model.AccessToken) {
	o.Token = token
}

func (o *Permissions) GetDoer() *user_model.User {
	return o.Doer
}

func (o *Permissions) SetDoer(doer *user_model.User) {
	o.Doer = doer
}

func (o *Permissions) GetPublicOnly() bool {
	return o.PublicOnly
}

func (o *Permissions) SetPublicOnly(publicOnly bool) {
	o.PublicOnly = publicOnly
}

func (o *Permissions) GetReducer() authz.AuthorizationReducer {
	return o.Reducer
}

func (o *Permissions) SetReducer(reducer authz.AuthorizationReducer) {
	o.Reducer = reducer
}
