// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	auth_model "forgejo.org/models/auth"
	"forgejo.org/models/db"
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

// Validate and insert a new authorized integration.
func InsertAuthorizedIntegration(ctx context.Context, ai *auth_model.AuthorizedIntegration, repoResources []*auth_model.AuthorizedIntegResourceRepo) error {
	ai.Name = strings.TrimSpace(ai.Name)
	ai.Description = strings.TrimSpace(ai.Description)

	if err := ValidateAuthorizedIntegration(ai, repoResources); err != nil {
		return err
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		if err := auth_model.InsertAuthorizedIntegration(ctx, ai); err != nil {
			return err
		}
		if !ai.ResourceAllRepos {
			if err := auth_model.InsertAuthorizedIntegrationResourceRepos(ctx, ai.ID, repoResources); err != nil {
				return err
			}
		}
		return nil
	})
}

func UpdateAuthorizedIntegration(ctx context.Context, ai *auth_model.AuthorizedIntegration, repoResources []*auth_model.AuthorizedIntegResourceRepo) error {
	ai.Name = strings.TrimSpace(ai.Name)
	ai.Description = strings.TrimSpace(ai.Description)

	if err := ValidateAuthorizedIntegration(ai, repoResources); err != nil {
		return err
	}

	return db.WithTx(ctx, func(ctx context.Context) error {
		if err := auth_model.UpdateAuthorizedIntegration(ctx, ai); err != nil {
			return err
		}
		return auth_model.UpdateAuthorizedIntegrationResourceRepos(ctx, ai.ID, repoResources)
	})
}
