// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"

	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
)

// FederatedRepository represents an ActivityPub Actor for federated repository.
type FederatedRepository struct {
	ID        int64  `xorm:"pk autoincr"`
	OwnerID   int64  `xorm:"owner_id NOT NULL INDEX REFERENCES(federated_user, id)"`
	ObjectID  ap.ID  `xorm:"object_id NOT NULL"`
	Name      string `xorm:"name NOT NULL"`
	Summary   string `xorm:"summary NOT NULL"`
	Inbox     ap.IRI `xorm:"inbox"`
	Outbox    ap.IRI `xorm:"outbox"`
	Followers ap.IRI `xorm:"followers"`
	Team      ap.IRI `xorm:"team"`
}

// Validate performs checks to validate the `FederatedRepository`.
func (repo FederatedRepository) Validate() []string {
	var (
		res      []string
		emptyIRI ap.IRI
	)

	res = append(res, validateID(repo.ID, "ID")...)
	res = append(res, validateID(repo.OwnerID, "OwnerID")...)
	res = append(res, validation.ValidateIRI(repo.ObjectID, "ObjectID")...)
	if repo.Inbox != emptyIRI {
		res = append(res, validation.ValidateIRI(repo.Inbox, "Inbox")...)
	}
	if repo.Outbox != emptyIRI {
		res = append(res, validation.ValidateIRI(repo.Outbox, "Outbox")...)
	}
	if repo.Followers != emptyIRI {
		res = append(res, validation.ValidateIRI(repo.Followers, "Followers")...)
	}
	if repo.Team != emptyIRI {
		res = append(res, validation.ValidateIRI(repo.Team, "Team")...)
	}

	return res
}

// validateID checks that the `FederatedRepository` ID is non-negative.
func validateID(id int64, field string) []string {
	var res []string

	if id < 0 {
		res = []string{fmt.Sprintf("invalid FederatedRepository %v: %v", field, id)}
	}

	return res
}
