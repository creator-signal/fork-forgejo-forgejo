// Copyright 2017 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package repo

import (
	"context"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/timeutil"
)

// WatchMode specifies what kind of watch the user has on a repository
type WatchMode int8

const (
	// WatchModeNone don't watch
	WatchModeNone WatchMode = iota // 0
	// WatchModeNormal watch repository (from other sources)
	WatchModeNormal // 1
	// WatchModeDont explicit don't auto-watch
	WatchModeDont // 2
	// WatchModeAuto watch repository (from AutoWatchOnChanges)
	WatchModeAuto // 3
)

// WatchEventType is a bitmask for granular notification event filtering
type WatchEventType int64

const (
	// WatchEventIssues triggers notifications for issue events
	WatchEventIssues WatchEventType = 1 << iota // 1
	// WatchEventPullRequests triggers notifications for pull request events
	WatchEventPullRequests // 2
	// WatchEventReleases triggers notifications for release events
	WatchEventReleases // 4
)

const (
	// WatchEventAll is a convenience constant for all event types
	WatchEventAll WatchEventType = WatchEventIssues | WatchEventPullRequests | WatchEventReleases // 7
)

// WatchesIssues returns true if this watch includes issue notifications
func (e WatchEventType) WatchesIssues() bool {
	return e&WatchEventIssues != 0
}

// WatchesPullRequests returns true if this watch includes pull request notifications
func (e WatchEventType) WatchesPullRequests() bool {
	return e&WatchEventPullRequests != 0
}

// WatchesReleases returns true if this watch includes release notifications
func (e WatchEventType) WatchesReleases() bool {
	return e&WatchEventReleases != 0
}

// Watch is connection request for receiving repository notification.
type Watch struct {
	ID          int64              `xorm:"pk autoincr"`
	UserID      int64              `xorm:"UNIQUE(watch)"`
	RepoID      int64              `xorm:"UNIQUE(watch)"`
	Mode        WatchMode          `xorm:"SMALLINT NOT NULL DEFAULT 1"`
	WatchEvents WatchEventType     `xorm:"BIGINT NOT NULL DEFAULT 7"`
	CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
}

// GetWatchEvents returns the watch events bitmask, defaulting to all events if not set
func (w *Watch) GetWatchEvents() WatchEventType {
	if w.WatchEvents == 0 {
		return WatchEventAll
	}
	return w.WatchEvents
}

func init() {
	db.RegisterModel(new(Watch))
}

// GetWatch gets what kind of subscription a user has on a given repository; returns dummy record if none found
func GetWatch(ctx context.Context, userID, repoID int64) (Watch, error) {
	watch := Watch{UserID: userID, RepoID: repoID}
	has, err := db.GetEngine(ctx).Get(&watch)
	if err != nil {
		return watch, err
	}
	if !has {
		watch.Mode = WatchModeNone
	}
	return watch, nil
}

// IsWatchMode Decodes watchability of WatchMode
func IsWatchMode(mode WatchMode) bool {
	return mode != WatchModeNone && mode != WatchModeDont
}

// IsWatching checks if user has watched given repository.
func IsWatching(ctx context.Context, userID, repoID int64) bool {
	watch, err := GetWatch(ctx, userID, repoID)
	return err == nil && IsWatchMode(watch.Mode)
}

func watchRepoMode(ctx context.Context, watch Watch, mode WatchMode) (err error) {
	if watch.Mode == mode {
		return nil
	}
	if mode == WatchModeAuto && (watch.Mode == WatchModeDont || IsWatchMode(watch.Mode)) {
		// Don't auto watch if already watching or deliberately not watching
		return nil
	}

	hadrec := watch.Mode != WatchModeNone
	needsrec := mode != WatchModeNone
	repodiff := 0

	if IsWatchMode(mode) && !IsWatchMode(watch.Mode) {
		repodiff = 1
	} else if !IsWatchMode(mode) && IsWatchMode(watch.Mode) {
		repodiff = -1
	}

	watch.Mode = mode

	if !hadrec && needsrec {
		watch.Mode = mode
		if err = db.Insert(ctx, watch); err != nil {
			return err
		}
	} else if needsrec {
		watch.Mode = mode
		if _, err := db.GetEngine(ctx).ID(watch.ID).AllCols().Update(watch); err != nil {
			return err
		}
	} else if _, err = db.DeleteByID[Watch](ctx, watch.ID); err != nil {
		return err
	}
	if repodiff != 0 {
		_, err = db.GetEngine(ctx).Exec("UPDATE `repository` SET num_watches = num_watches + ? WHERE id = ?", repodiff, watch.RepoID)
	}
	return err
}

// WatchRepoMode watch repository in specific mode.
func WatchRepoMode(ctx context.Context, userID, repoID int64, mode WatchMode) (err error) {
	var watch Watch
	if watch, err = GetWatch(ctx, userID, repoID); err != nil {
		return err
	}
	return watchRepoMode(ctx, watch, mode)
}

// WatchRepo watch or unwatch repository.
func WatchRepo(ctx context.Context, userID, repoID int64, doWatch bool) (err error) {
	var watch Watch
	if watch, err = GetWatch(ctx, userID, repoID); err != nil {
		return err
	}
	if !doWatch && watch.Mode == WatchModeAuto {
		err = watchRepoMode(ctx, watch, WatchModeDont)
	} else if !doWatch {
		err = watchRepoMode(ctx, watch, WatchModeNone)
	} else {
		err = watchRepoMode(ctx, watch, WatchModeNormal)
	}
	return err
}

// GetWatchers returns all watchers of given repository.
func GetWatchers(ctx context.Context, repoID int64) ([]*Watch, error) {
	watches := make([]*Watch, 0, 10)
	return watches, db.GetEngine(ctx).Where("`watch`.repo_id=?", repoID).
		And("`watch`.mode<>?", WatchModeDont).
		And("`user`.is_active=?", true).
		And("`user`.prohibit_login=?", false).
		Join("INNER", "`user`", "`user`.id = `watch`.user_id").
		Find(&watches)
}

// GetRepoWatchersIDs returns IDs of watchers for a given repo ID
// but avoids joining with `user` for performance reasons
// User permissions must be verified elsewhere if required
func GetRepoWatchersIDs(ctx context.Context, repoID int64) ([]int64, error) {
	ids := make([]int64, 0, 64)
	return ids, db.GetEngine(ctx).Table("watch").
		Where("watch.repo_id=?", repoID).
		And("watch.mode<>?", WatchModeDont).
		Select("user_id").
		Find(&ids)
}

// GetRepoWatchersIDsForEvent returns IDs of watchers for a given repo ID
// filtered by the specified event type (Issues, PRs, Releases).
// This enables granular notifications where users can choose which events to watch.
// User permissions must be verified elsewhere if required.
// For backwards compatibility, watch_events=0 is treated as "all events".
func GetRepoWatchersIDsForEvent(ctx context.Context, repoID int64, eventType WatchEventType) ([]int64, error) {
	ids := make([]int64, 0, 64)
	return ids, db.GetEngine(ctx).Table("watch").
		Where("watch.repo_id=?", repoID).
		And("watch.mode<>?", WatchModeDont).
		And("(watch.watch_events = 0 OR watch.watch_events & ? != 0)", eventType).
		Select("user_id").
		Find(&ids)
}

// UpdateWatchEvents updates the event filter bitmask for a user's watch on a repository.
// If the user is not currently watching, this will start watching with the specified events.
// If events is 0, this will unwatch the repository.
func UpdateWatchEvents(ctx context.Context, userID, repoID int64, events WatchEventType) error {
	watch, err := GetWatch(ctx, userID, repoID)
	if err != nil {
		return err
	}

	if events == 0 {
		return WatchRepo(ctx, userID, repoID, false)
	}

	wasWatching := IsWatchMode(watch.Mode)

	// If not currently watching, start watching
	if !wasWatching {
		watch.Mode = WatchModeNormal
		watch.WatchEvents = events
		if err := db.Insert(ctx, watch); err != nil {
			return err
		}
		_, err = db.GetEngine(ctx).Exec("UPDATE `repository` SET num_watches = num_watches + 1 WHERE id = ?", repoID)
		return err
	}

	watch.WatchEvents = events
	_, err = db.GetEngine(ctx).ID(watch.ID).Cols("watch_events").Update(&watch)
	return err
}

// WatchRepoWithEvents starts watching a repository with specific event types.
func WatchRepoWithEvents(ctx context.Context, userID, repoID int64, events WatchEventType) error {
	watch, err := GetWatch(ctx, userID, repoID)
	if err != nil {
		return err
	}

	wasWatching := IsWatchMode(watch.Mode)

	if !wasWatching {
		watch.Mode = WatchModeNormal
		watch.WatchEvents = events
		if err := db.Insert(ctx, watch); err != nil {
			return err
		}
		_, err = db.GetEngine(ctx).Exec("UPDATE `repository` SET num_watches = num_watches + 1 WHERE id = ?", repoID)
		return err
	}

	watch.WatchEvents = events
	_, err = db.GetEngine(ctx).ID(watch.ID).Cols("watch_events").Update(&watch)
	return err
}

// GetRepoWatchers returns range of users watching given repository.
func GetRepoWatchers(ctx context.Context, repoID int64, opts db.ListOptions) ([]*user_model.User, error) {
	sess := db.GetEngine(ctx).Where("watch.repo_id=?", repoID).
		Join("LEFT", "watch", "`user`.id=`watch`.user_id").
		And("`watch`.mode<>?", WatchModeDont)
	if opts.Page > 0 {
		sess = db.SetSessionPagination(sess, &opts)
		users := make([]*user_model.User, 0, opts.PageSize)

		return users, sess.Find(&users)
	}

	users := make([]*user_model.User, 0, 8)
	return users, sess.Find(&users)
}

// WatchIfAuto subscribes to repo if AutoWatchOnChanges is set
func WatchIfAuto(ctx context.Context, userID, repoID int64, isWrite bool) error {
	if !isWrite || !setting.Service.AutoWatchOnChanges {
		return nil
	}
	watch, err := GetWatch(ctx, userID, repoID)
	if err != nil {
		return err
	}
	if watch.Mode != WatchModeNone {
		return nil
	}
	return watchRepoMode(ctx, watch, WatchModeAuto)
}

// WatchIfAutoWithEvents subscribes to repo with specific events if AutoWatchOnChanges is set
func WatchIfAutoWithEvents(ctx context.Context, userID, repoID int64, isWrite bool, events WatchEventType) error {
	if !isWrite || !setting.Service.AutoWatchOnChanges {
		return nil
	}
	watch, err := GetWatch(ctx, userID, repoID)
	if err != nil {
		return err
	}
	if watch.Mode != WatchModeNone {
		return nil
	}

	// Set watch events before creating the watch
	watch.WatchEvents = events

	return watchRepoMode(ctx, watch, WatchModeAuto)
}

// UnwatchRepos will unwatch the user from all given repositories.
func UnwatchRepos(ctx context.Context, userID int64, repoIDs []int64) error {
	_, err := db.GetEngine(ctx).Where("user_id=?", userID).In("repo_id", repoIDs).Delete(&Watch{})
	return err
}
