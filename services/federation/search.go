// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package federation

import (
	"context"

	"forgejo.org/modules/webfinger"
)

type WebfingerSearch struct{}

func (WebfingerSearch) Search(ctx context.Context, actor string) (any, error) {
	jrd, err := webfinger.Query(ctx, actor)
	if err != nil {
		return nil, err
	}

	profileActivity, err := jrd.GetProfileActivity()
	if err != nil {
		return nil, err
	}

	user, _, _, err := FindOrCreateFederatedUser(ctx, profileActivity.ActivityLocation.String())
	return user, err
}
