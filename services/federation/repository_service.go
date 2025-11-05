// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"

	forgefed "forgejo.org/modules/forgefed"
	ap "github.com/go-ap/activitypub"
)

// ProcessRepositoryInbox handles requests for a federated repository inbox.
func ProcessRepositoryInbox(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	switch activity.Type {
	case ap.LikeType:
		return ProcessLikeActivity(ctx, activity, repositoryID)
	case ap.OfferType:
		if !activity.Object.IsObject() {
			return ServiceResult{}, NewErrNotAcceptablef("Invalid repository offer activity object: %v", activity.Object)
		}

		switch activity.Object.GetType() {
		case forgefed.TicketType:
			return ProcessInboxTicketActivity(ctx, activity, repositoryID)
		default:
			return ServiceResult{}, NewErrNotAcceptablef("Unsupported repository offer activity object type: %v", activity.Object.GetType())

		}
	default:
		return ServiceResult{}, NewErrNotAcceptablef("Not a repository activity: %v", activity.Type)
	}
}

// ProcessRepositoryOutbox handles requests for a federated repository outbox.
func ProcessRepositoryOutbox(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	return ProcessOutboxActivity(ctx, activity, repositoryID)
}
