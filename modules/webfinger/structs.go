// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webfinger

import (
	"net/url"

	"forgejo.org/modules/log"
	"forgejo.org/modules/optional"
)

type ProfileActivity struct {
	ProfilePage      optional.Option[*url.URL]
	ActivityLocation *url.URL
}

func (jrd JRD) GetProfileActivity() (*ProfileActivity, error) {
	profilePage := optional.None[*url.URL]()
	activityLocation := optional.None[*url.URL]()

	for _, v := range jrd.Links {
		if v.Type == "text/html" || v.Rel == "http://webfinger.net/rel/profile-page" {
			if activityLocation.Has() {
				log.Warn("Already found profile page in Links, ignoring")
				continue
			}

			url, err := url.Parse(v.Href)
			if err != nil {
				return nil, err
			}

			profilePage = optional.Some(url)
		} else if v.Type == "application/activity+json" {
			if activityLocation.Has() {
				log.Warn("Already found application/activity+json in Links, ignoring")
				continue
			}

			url, err := url.Parse(v.Href)
			if err != nil {
				return nil, err
			}

			activityLocation = optional.Some(url)
		}
	}

	if !activityLocation.Has() {
		return nil, MissingActivityEndpoint{}
	}

	profileActivity := ProfileActivity{
		ProfilePage:      profilePage,
		ActivityLocation: activityLocation.Value(),
	}

	return &profileActivity, nil
}
