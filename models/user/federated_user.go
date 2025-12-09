// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package user

import (
	"context"
	"fmt"
	"strings"

	"forgejo.org/models/federation_key"
	"forgejo.org/models/forgefed"
	"forgejo.org/modules/validation"
)

type FederatedUser struct {
	ID                    int64  `xorm:"pk autoincr"`
	UserID                int64  `xorm:"NOT NULL INDEX user_id"`
	ExternalID            string `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	FederationHostID      int64  `xorm:"UNIQUE(federation_user_mapping) NOT NULL"`
	InboxPath             string
	NormalizedOriginalURL string // This field is just to keep original information. Pls. do not use for search or as ID!
}

func NewFederatedUser(userID int64, externalID string, federationHostID int64, inboxPath, normalizedOriginalURL string) (FederatedUser, error) {
	result := FederatedUser{
		UserID:                userID,
		ExternalID:            externalID,
		FederationHostID:      federationHostID,
		InboxPath:             inboxPath,
		NormalizedOriginalURL: normalizedOriginalURL,
	}
	if valid, err := validation.IsValid(result); !valid {
		return FederatedUser{}, err
	}
	return result, nil
}

func (federatedUser FederatedUser) Validate() []string {
	var result []string
	result = append(result, validation.ValidateNotEmpty(federatedUser.UserID, "UserID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.ExternalID, "ExternalID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.FederationHostID, "FederationHostID")...)
	result = append(result, validation.ValidateNotEmpty(federatedUser.InboxPath, "InboxPath")...)
	return result
}

// ValidateKeyID checks that the provided ActivityPub keyID matches the user.
func (federatedUser FederatedUser) ValidateKeyID(ctx context.Context, keyID federation_key.KeyID) error {
	host, err := forgefed.GetFederationHost(ctx, federatedUser.FederationHostID)
	if err != nil {
		return err
	}

	return federatedUser.ValidateKeyIDWithHost(keyID, host)
}

// ValidateKeyIDWithHost checks that the provided ActivityPub keyID matches the user.
func (federatedUser FederatedUser) ValidateKeyIDWithHost(keyID federation_key.KeyID, host *forgefed.FederationHost) error {
	if host == nil {
		return fmt.Errorf("nil federation host")
	}

	if federatedUser.FederationHostID != host.ID {
		return fmt.Errorf("invalid host ID, have: %v, expected: %v", host.ID, federatedUser.FederationHostID)
	}

	keyURL, err := keyID.IRI().URL()
	if err != nil {
		return err
	}

	if err = host.ValidateKeyID(keyID); err != nil {
		return err
	}

	userPath, found := strings.CutSuffix(federatedUser.InboxPath, "/inbox")
	if !found {
		if userPath, found = strings.CutSuffix(federatedUser.InboxPath, "/inbox/"); !found {
			return fmt.Errorf("invalid user inbox path: %v", federatedUser.InboxPath)
		}
	}

	if keyURL.Path != userPath {
		return fmt.Errorf("invalid user key ID path, key ID path: %v, user path: %v", keyURL.Path, userPath)
	}

	return nil
}
