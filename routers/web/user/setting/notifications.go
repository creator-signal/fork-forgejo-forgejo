// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package setting

import (
	"errors"
	"net/http"
	"strconv"

	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/base"
	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
	"forgejo.org/modules/setting"
	"forgejo.org/services/context"
	user_service "forgejo.org/services/user"
)

const tplSettingsNotifications base.TplName = "user/settings/notifications"

// Notifications render user's notification settings
func Notifications(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.notifications")
	ctx.Data["PageIsSettingsNotifications"] = true

	loadNotificationsData(ctx)

	ctx.HTML(http.StatusOK, tplSettingsNotifications)
}

// NotificationsPost handles POST requests for notification settings
func NotificationsPost(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("settings.notifications")
	ctx.Data["PageIsSettingsNotifications"] = true

	if ctx.FormString("_method") == "EMAIL_PREFERENCE" {
		preference := ctx.FormString("preference")
		if preference != user_model.EmailNotificationsEnabled &&
			preference != user_model.EmailNotificationsOnMention &&
			preference != user_model.EmailNotificationsDisabled &&
			preference != user_model.EmailNotificationsAndYourOwn {
			log.Error("Email notifications preference returned unrecognized option %s: %s", preference, ctx.Doer.Name)
			ctx.ServerError("SetEmailPreference", errors.New("option unrecognized"))
			return
		}
		opts := &user_service.UpdateOptions{
			EmailNotificationsPreference: optional.Some(preference),
		}
		if err := user_service.UpdateUser(ctx, ctx.Doer, opts); err != nil {
			log.Error("Set Email Notifications failed: %v", err)
			ctx.ServerError("UpdateUser", err)
			return
		}
		log.Trace("Email notifications preference made %s: %s", preference, ctx.Doer.Name)
		ctx.Flash.Success(ctx.Tr("settings.email_preference_set_success"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/notifications")
		return
	}

	// Set Auto Watch Repos
	if ctx.FormString("_method") == "AUTO_WATCH" {
		settings := map[string]bool{
			user_model.SettingsKeyAutoWatchOnCreate:     ctx.FormBool("auto_watch_on_create"),
			user_model.SettingsKeyAutoWatchOnAccess:     ctx.FormBool("auto_watch_on_access"),
			user_model.SettingsKeyAutoWatchOnContribute: ctx.FormBool("auto_watch_on_contribute"),
		}
		for key, enabled := range settings {
			value := "false"
			if enabled {
				value = "true"
			}
			if err := user_model.SetUserSetting(ctx, ctx.Doer.ID, key, value); err != nil {
				log.Error("Set %s failed: %v", key, err)
				ctx.ServerError("SetUserSetting", err)
				return
			}
		}
		log.Trace("Auto watch settings updated for user: %s", ctx.Doer.Name)
		ctx.Flash.Success(ctx.Tr("settings.auto_watch_repos.success"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/notifications")
		return
	}

	// Set Default Watch Events
	if ctx.FormString("_method") == "WATCH_DEFAULTS" {
		var events repo_model.WatchEventType
		if ctx.FormBool("watch_issues") {
			events |= repo_model.WatchEventIssues
		}
		if ctx.FormBool("watch_pulls") {
			events |= repo_model.WatchEventPullRequests
		}
		if ctx.FormBool("watch_releases") {
			events |= repo_model.WatchEventReleases
		}

		if err := user_model.SetUserSetting(ctx, ctx.Doer.ID, user_model.SettingsKeyDefaultWatchEvents, strconv.FormatInt(int64(events), 10)); err != nil {
			log.Error("Set default watch events failed: %v", err)
			ctx.ServerError("SetUserSetting", err)
			return
		}
		log.Trace("Default watch events set to %d: %s", events, ctx.Doer.Name)
		ctx.Flash.Success(ctx.Tr("settings.default_watch_events.success"))
		ctx.Redirect(setting.AppSubURL + "/user/settings/notifications")
		return
	}

	loadNotificationsData(ctx)
	ctx.HTML(http.StatusOK, tplSettingsNotifications)
}

func loadNotificationsData(ctx *context.Context) {
	ctx.Data["EmailNotificationsPreference"] = ctx.Doer.EmailNotificationsPreference
	ctx.Data["EnableNotifyMail"] = setting.Service.EnableNotifyMail

	// Load auto-watch settings (each defaults to the appropriate instance setting)
	// Auto-watch on create - defaults to AUTO_WATCH_NEW_REPOS
	autoWatchOnCreate := setting.Service.AutoWatchNewRepos
	if val, err := user_model.GetUserSetting(ctx, ctx.Doer.ID, user_model.SettingsKeyAutoWatchOnCreate); err == nil && val != "" {
		autoWatchOnCreate = val == "true"
	}
	ctx.Data["AutoWatchOnCreate"] = autoWatchOnCreate

	// Auto-watch on access - defaults to AUTO_WATCH_NEW_REPOS
	autoWatchOnAccess := setting.Service.AutoWatchNewRepos
	if val, err := user_model.GetUserSetting(ctx, ctx.Doer.ID, user_model.SettingsKeyAutoWatchOnAccess); err == nil && val != "" {
		autoWatchOnAccess = val == "true"
	}
	ctx.Data["AutoWatchOnAccess"] = autoWatchOnAccess

	// Auto-watch on contribute - defaults to AUTO_WATCH_ON_CHANGES
	autoWatchOnContribute := setting.Service.AutoWatchOnChanges
	if val, err := user_model.GetUserSetting(ctx, ctx.Doer.ID, user_model.SettingsKeyAutoWatchOnContribute); err == nil && val != "" {
		autoWatchOnContribute = val == "true"
	}
	ctx.Data["AutoWatchOnContribute"] = autoWatchOnContribute

	defaultWatchEvents := repo_model.WatchEventAll
	if val, err := user_model.GetUserSetting(ctx, ctx.Doer.ID, user_model.SettingsKeyDefaultWatchEvents); err == nil && val != "" {
		if events, err := strconv.ParseInt(val, 10, 64); err == nil && events > 0 {
			defaultWatchEvents = repo_model.WatchEventType(events)
		}
	}
	ctx.Data["DefaultWatchIssues"] = defaultWatchEvents.WatchesIssues()
	ctx.Data["DefaultWatchPulls"] = defaultWatchEvents.WatchesPullRequests()
	ctx.Data["DefaultWatchReleases"] = defaultWatchEvents.WatchesReleases()
}
