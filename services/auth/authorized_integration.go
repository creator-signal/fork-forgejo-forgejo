// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"errors"
	"fmt"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/services/authz"
)

var ErrAuthorizedIntegrationBadUI = errors.New("invalid authorized integration UI")

// Validate that an authorized integration's state is valid for creation.  For example, that it doesn't have a
// conflicting set of resources (public-only and specific repositories), and other similar checks.
func ValidateAuthorizedIntegration(ai *auth_model.AuthorizedIntegration, repoResources []*auth_model.AuthorizedIntegResourceRepo) error {
	switch ai.UI {
	case auth_model.AuthorizedIntegrationUIGeneric,
		auth_model.AuthorizedIntegrationUIForgejoActionsLocal:
		break
	default:
		return fmt.Errorf("%w: invalid UI: %q", ErrAuthorizedIntegrationBadUI, ai.UI)
	}
	return authz.ValidateRepositoryResource(ai.ResourceAllRepos, ai.Scope, len(repoResources))
}
