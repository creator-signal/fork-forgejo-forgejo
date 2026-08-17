// Copyright 2026 Forgejo Authors
// SPDX-License-Identifier: MIT

package context

import (
	"context"
	"net/http"

	apiv1_permissions "forgejo.org/routers/api/v1/permissions"
	f3_permissions_errors "forgejo.org/services/f3/permissions/errors"
)

type key int

type F3 struct {
	sendNotifications bool
	mirrorID          int64
	apiv1_permissions.Permissions
}

func New() *F3 {
	return &F3{}
}

func (o *F3) SetSendNotifications(sendNotifications bool) *F3 {
	o.sendNotifications = sendNotifications
	return o
}

func (o *F3) GetSendNotifications() bool {
	return o.sendNotifications
}

func (o *F3) SetMirrorID(mirrorID int64) *F3 {
	o.mirrorID = mirrorID
	return o
}

func (o *F3) GetMirrorID() int64 {
	return o.mirrorID
}

func (o *F3) GetError() error {
	if o.GetStatus() == 0 {
		return nil
	}
	switch o.GetStatus() {
	case http.StatusNotFound:
		return f3_permissions_errors.NewNotFound(o.GetMessage())
	case http.StatusUnauthorized:
		return f3_permissions_errors.NewUnauthorized(o.GetMessage())
	case http.StatusForbidden:
		return f3_permissions_errors.NewForbidden(o.GetMessage())
	default:
		return f3_permissions_errors.NewServer(o.GetMessage())
	}
}

var _ apiv1_permissions.Context = &F3{}

const (
	f3Key key = iota + 1
)

func WithF3(ctx context.Context, f3 *F3) context.Context {
	f3.SetContext(ctx)
	return context.WithValue(ctx, f3Key, f3)
}

func Get(ctx context.Context) *F3 {
	if value := ctx.Value(f3Key); value != nil {
		return value.(*F3)
	}
	return nil
}
