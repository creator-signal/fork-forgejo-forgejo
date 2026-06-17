// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

// Package audit defines the storage model for the security audit log: a record
// of security-relevant events such as logins, access-token creation and changes
// to account security settings.
package audit

import (
	"context"

	"forgejo.org/models/db"
	"forgejo.org/modules/timeutil"
)

// Action identifies the type of a recorded audit event. Values are stable
// machine identifiers and must not be changed once released.
type Action string

const (
	// UserLogin is recorded when a user successfully signs in.
	UserLogin Action = "user_login"
	// UserAccessTokenAdd is recorded when a personal access token is created.
	UserAccessTokenAdd Action = "user_access_token_add"
	// UserOAuth2ApplicationAdd is recorded when an OAuth2 application is registered.
	UserOAuth2ApplicationAdd Action = "user_oauth2_application_add"
	// UserTwoFactorEnable is recorded when TOTP two-factor authentication is enabled.
	UserTwoFactorEnable Action = "user_two_factor_enable"
	// UserTwoFactorDisable is recorded when TOTP two-factor authentication is disabled.
	UserTwoFactorDisable Action = "user_two_factor_disable"
	// UserPasswordChange is recorded when a user changes their password.
	UserPasswordChange Action = "user_password_change"
	// UserEmailAdd is recorded when a user adds an email address.
	UserEmailAdd Action = "user_email_add"
	// UserEmailPrimaryChange is recorded when a user changes their primary email address.
	UserEmailPrimaryChange Action = "user_email_primary_change"
)

// Event is a single security-relevant action recorded in the audit log.
//
// DoerName and TargetName are denormalized on purpose: an audit trail must
// remain readable even after the referenced user or object has been deleted.
type Event struct {
	ID          int64  `xorm:"pk autoincr"`
	Action      Action `xorm:"INDEX NOT NULL"`
	DoerID      int64  `xorm:"INDEX"`
	DoerName    string
	ScopeType   string
	ScopeID     int64
	TargetType  string
	TargetID    int64
	TargetName  string
	Message     string `xorm:"TEXT"`
	IPAddress   string
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
}

func init() {
	db.RegisterModel(new(Event))
}

// TableName specifies the database table backing audit events.
func (Event) TableName() string {
	return "audit_event"
}

// InsertEvent persists a single audit event.
func InsertEvent(ctx context.Context, e *Event) error {
	return db.Insert(ctx, e)
}

// CountEvents returns the total number of recorded audit events.
func CountEvents(ctx context.Context) int64 {
	count, _ := db.GetEngine(ctx).Count(new(Event))
	return count
}

// Events returns audit events for the given page, newest first.
func Events(ctx context.Context, page, pageSize int) ([]*Event, error) {
	events := make([]*Event, 0, pageSize)
	return events, db.GetEngine(ctx).
		Limit(pageSize, (page-1)*pageSize).
		Desc("created_unix").
		Find(&events)
}
