// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package audit records security-relevant events (logins, access-token
// creation, security-setting changes, ...) to a queryable audit log.
// Recording is a no-op unless auditing is enabled via the [audit] config
// section, and a recording failure never blocks the audited action.
package audit

import (
	"context"

	audit_model "forgejo.org/models/audit"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
)

// Audit actions are re-exported so callers only need to import this package.
const (
	UserLogin                = audit_model.UserLogin
	UserAccessTokenAdd       = audit_model.UserAccessTokenAdd
	UserOAuth2ApplicationAdd = audit_model.UserOAuth2ApplicationAdd
	UserTwoFactorEnable      = audit_model.UserTwoFactorEnable
	UserTwoFactorDisable     = audit_model.UserTwoFactorDisable
	UserPasswordChange       = audit_model.UserPasswordChange
	UserEmailAdd             = audit_model.UserEmailAdd
	UserEmailPrimaryChange   = audit_model.UserEmailPrimaryChange
)

// TypeDescriptor identifies the object an audit event is about (for example the
// access token that was created). The zero value means "no specific target".
type TypeDescriptor struct {
	Type string
	ID   int64
	Name string
}

// Record persists a security-relevant audit event. It is a no-op unless
// auditing is enabled. A persistence failure is logged but never returned, so
// auditing can never block or fail the action being audited.
func Record(ctx context.Context, action audit_model.Action, doer *user_model.User, ipAddress string, target TypeDescriptor, message string) {
	if !setting.Audit.Enabled {
		return
	}

	e := &audit_model.Event{
		Action:     action,
		IPAddress:  ipAddress,
		TargetType: target.Type,
		TargetID:   target.ID,
		TargetName: target.Name,
		Message:    message,
	}
	if doer != nil {
		e.DoerID = doer.ID
		e.DoerName = doer.Name
		// Security events currently always concern the doer's own account.
		e.ScopeType = "user"
		e.ScopeID = doer.ID
	}

	if err := audit_model.InsertEvent(ctx, e); err != nil {
		log.Error("audit.Record: failed to persist %q event: %v", action, err)
		return
	}

	log.Info("audit: %s by %q from %s: %s", action, e.DoerName, ipAddress, message)
}
