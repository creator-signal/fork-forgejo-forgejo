// Copyright 2019 The Gitea Authors. All rights reserved.
// Copyright 2018 Jonas Franz. All rights reserved.
// SPDX-License-Identifier: MIT

package allowlist

import (
	"errors"
	"net/url"
	"path/filepath"
	"strings"

	"forgejo.org/models"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/hostmatcher"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
)

var (
	allowList *hostmatcher.HostMatchList
	blockList *hostmatcher.HostMatchList
)

// IsPushMirrorURLAllowed checks if an URL is allowed to be pushed to.
func IsPushMirrorURLAllowed(remoteURL string, doer *user_model.User) (hostmatcher.ManualDialContext, error) {
	return isURLAllowed(remoteURL, doer, true)
}

// IsMigrateURLAllowed checks if an URL is allowed to be migrated from.
func IsMigrateURLAllowed(remoteURL string, doer *user_model.User) (hostmatcher.ManualDialContext, error) {
	return isURLAllowed(remoteURL, doer, false)
}

func isURLAllowed(remoteURL string, doer *user_model.User, isPushMirror bool) (hostmatcher.ManualDialContext, error) {
	// Remote address can be HTTP/HTTPS/Git URL or local path.
	u, err := url.Parse(remoteURL)
	if err != nil {
		return nil, &models.ErrInvalidCloneAddr{IsURLError: true, Host: remoteURL}
	}

	if u.Scheme == "file" || u.Scheme == "" {
		if !doer.CanImportLocal() {
			return nil, &models.ErrInvalidCloneAddr{Host: "<LOCAL_FILESYSTEM>", IsPermissionDenied: true, LocalPath: true}
		}
		isAbs := filepath.IsAbs(u.Host + u.Path)
		if !isAbs {
			return nil, &models.ErrInvalidCloneAddr{Host: "<LOCAL_FILESYSTEM>", IsInvalidPath: true, LocalPath: true}
		}
		isDir, err := util.IsDir(u.Host + u.Path)
		if err != nil {
			log.Error("Unable to check if %s is a directory: %v", u.Host+u.Path, err)
			return nil, err
		}
		if !isDir {
			return nil, &models.ErrInvalidCloneAddr{Host: "<LOCAL_FILESYSTEM>", IsInvalidPath: true, LocalPath: true}
		}

		return hostmatcher.NewNilManualDialContext(), nil
	}

	if u.Scheme == "git" && u.Port() != "" && (strings.Contains(remoteURL, "%0d") || strings.Contains(remoteURL, "%0a")) {
		return nil, &models.ErrInvalidCloneAddr{Host: u.Host, IsURLError: true}
	}

	if u.Opaque != "" || u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "git" && u.Scheme != "ssh" || (!isPushMirror && u.Scheme == "ssh") {
		return nil, &models.ErrInvalidCloneAddr{Host: u.Host, IsProtocolInvalid: true, IsPermissionDenied: true, IsURLError: true}
	}

	mdc := hostmatcher.NewRemoteManualDialContext(u, allowList, blockList)
	if err := mdc.Check(); errors.Is(err, hostmatcher.ErrManualDialContextCheckFailed) {
		err = &models.ErrInvalidCloneAddr{Host: u.Hostname(), IsPermissionDenied: true}
		return nil, err
	}

	return mdc, nil
}

// Init migrations service
func Init() error {
	// TODO: maybe we can deprecate these legacy ALLOWED_DOMAINS/ALLOW_LOCALNETWORKS/BLOCKED_DOMAINS, use ALLOWED_HOST_LIST/BLOCKED_HOST_LIST instead

	blockList = hostmatcher.ParseSimpleMatchList("migrations.BLOCKED_DOMAINS", setting.Migrations.BlockedDomains)

	allowList = hostmatcher.ParseSimpleMatchList("migrations.ALLOWED_DOMAINS/ALLOW_LOCALNETWORKS", setting.Migrations.AllowedDomains)
	if allowList.IsEmpty() {
		// the default policy is that migration module can access external hosts
		allowList.AppendBuiltin(hostmatcher.MatchBuiltinExternal)
	}
	if setting.Migrations.AllowLocalNetworks {
		allowList.AppendBuiltin(hostmatcher.MatchBuiltinPrivate)
		allowList.AppendBuiltin(hostmatcher.MatchBuiltinLoopback)
	}
	// TODO: at the moment, if ALLOW_LOCALNETWORKS=false, ALLOWED_DOMAINS=domain.com, and domain.com has IP 127.0.0.1, then it's still allowed.
	// if we want to block such case, the private&loopback should be added to the blockList when ALLOW_LOCALNETWORKS=false

	return nil
}
