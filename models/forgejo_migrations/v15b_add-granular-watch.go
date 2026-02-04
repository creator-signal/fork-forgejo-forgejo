// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package forgejo_migrations

import (
	repo_model "forgejo.org/models/repo"

	"xorm.io/xorm"
)

func init() {
	registerMigration(&Migration{
		Description: "Add granular watch settings to repos.",
		Upgrade:     addGranularWatchColumnsAndDropModeColumn,
	})
}

func addGranularWatchColumnsAndDropModeColumn(x *xorm.Engine) error {
	type Watch struct {
		Source                     repo_model.WatchSource `xorm:"NOT NULL"`
		WatchSelectionIssues       bool                   `xorm:"NOT NULL"`
		WatchSelectionPullRequests bool                   `xorm:"NOT NULL"`
		WatchSelectionReleases     bool                   `xorm:"NOT NULL"`
	}
	if err := x.Sync(new(Watch)); err != nil {
		return err
	}

	// copy of old code //
	type WatchMode uint8
	const (
		// WatchModeNone don't watch
		// This means there is no Watch record in the db.
		// We never store this mode in the db and instead remove the record from the db.
		// Furthermore, this means there is a WatchMode for all combinations of user and repo.
		// We never go back to this state once we've been in a different state.
		WatchModeNone WatchMode = iota // 0
		// WatchModeNormal watch repository (from other sources)
		// This means the user explicitly chose to watch the repo.
		WatchModeNormal // 1
		// WatchModeDont explicit don't auto-watch
		// This means the user explicitly removed themselves as a watcher.
		// Then the AutoWatchOnChanges feature doesn't make the user a watcher when they push to the repo.
		WatchModeDont // 2
		// WatchModeAuto watch repository (from AutoWatchOnChanges)
		// This is used when the user pushed to the repo and setting.Service.AutoWatchOnChanges is true.
		// That way we can differentiate people explicitly watching the repo and people only watching it because of the AutoWatchOnChanges feature.
		WatchModeAuto // 3
	)
	// end copy of old code //

	_, err := x.Exec("UPDATE `watch` SET source = ? WHERE mode = ?", repo_model.WatchSourceAutomatic, WatchModeNone)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET source = ? WHERE mode = ?", repo_model.WatchSourceExplicit, WatchModeNormal)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET source = ? WHERE mode = ?", repo_model.WatchSourceExplicit, WatchModeDont)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET source = ? WHERE mode = ?", repo_model.WatchSourceAutomatic, WatchModeAuto)
	if err != nil {
		return err
	}

	_, err = x.Exec("UPDATE `watch` SET watch_selection_issues = ? WHERE mode = ? OR mode = ?", false, WatchModeNone, WatchModeDont)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET watch_selection_pull_requests = ? WHERE mode = ? OR mode = ?", false, WatchModeNone, WatchModeDont)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET watch_selection_releases = ? WHERE mode = ? OR mode = ?", false, WatchModeNone, WatchModeDont)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET watch_selection_issues = ? WHERE mode = ? OR mode = ?", true, WatchModeNormal, WatchModeAuto)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET watch_selection_pull_requests = ? WHERE mode = ? OR mode = ?", true, WatchModeNormal, WatchModeAuto)
	if err != nil {
		return err
	}
	_, err = x.Exec("UPDATE `watch` SET watch_selection_releases = ? WHERE mode = ? OR mode = ?", true, WatchModeNormal, WatchModeAuto)
	if err != nil {
		return err
	}

	_, err = x.Exec("ALTER TABLE watch DROP COLUMN `mode`")
	if err != nil {
		return err
	}
	return nil
}
