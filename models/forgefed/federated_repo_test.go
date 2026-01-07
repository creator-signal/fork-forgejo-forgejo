// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed_test

import (
	"testing"

	"forgejo.org/models/forgefed"
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/require"
)

func Test_RepositoryValidation(t *testing.T) {
	repoID := int64(1)
	ownerID := int64(2)
	objectID := "https://forgejo.org/user/repo"
	inbox := ap.IRI(objectID + "/inbox")
	outbox := ap.IRI(objectID + "/outbox")
	followers := ap.IRI(objectID + "/followers")
	team := ap.IRI(objectID + "/team")

	sut := forgefed.FederatedRepository{
		ObjectID:  ap.ID(objectID),
		Name:      "Test Repository",
		Summary:   "<p>A repository for ActivityPub test.</p>",
		ID:        repoID,
		OwnerID:   ownerID,
		Inbox:     inbox,
		Outbox:    outbox,
		Followers: followers,
		Team:      team,
	}

	_, err := validation.IsValid(sut)
	require.NoError(t, err, "expected valid FederatedRepository: %v", err)

	sut.ID = -1
	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository ID: %v", sut)

	sut.ID = repoID
	sut.OwnerID = -2

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository Owner ID: %v", sut)

	badIRI := ap.IRI("https://bad.url/%^*")

	sut.OwnerID = ownerID
	sut.Inbox = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository inbox: %v", sut)

	sut.Inbox = inbox
	sut.Outbox = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository outbox: %v", sut)

	sut.Outbox = outbox
	sut.Followers = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository followers: %v", sut)

	sut.Followers = followers
	sut.Team = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid FederatedRepository team: %v", sut)
}
