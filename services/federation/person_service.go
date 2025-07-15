// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"net/http"

	"forgejo.org/models/user"
	"forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
)

func ProcessPersonInbox(ctx context.Context, user *user.User, activity *ap.Activity) (ServiceResult, error) {
	switch activity.Type {
	case ap.CreateType:
		return processPersonInboxCreate(ctx, user, activity)
	case ap.FollowType:
		return processPersonFollow(ctx, user, activity)
	case ap.UndoType:
		return processPersonInboxUndo(ctx, user, activity)
	case ap.AcceptType:
		return processPersonInboxAccept(activity)
	}

	log.Error("Unsupported PersonInbox activity: %v", activity.Type)
	return ServiceResult{}, NewErrNotAcceptablef("Unsupported activity: %v", activity.Type)
}

func FollowRemoteActor(ctx *context_service.APIContext, localUser *user.User, actorURI string) error {
	_, federatedUser, federationHost, err := FindOrCreateFederatedUser(ctx.Base, actorURI)
	if err != nil {
		ctx.Error(http.StatusNotAcceptable, "Federated user not found", err)
		return err
	}

	followReq, err := forgefed.NewForgeFollow(localUser.APActorID(), actorURI)
	if err != nil {
		return err
	}

	payload, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).
		Marshal(followReq)
	if err != nil {
		return err
	}

	hostURL := federationHost.AsURL()
	return pendingQueue.Push(pendingQueueItem{
		InboxURL: hostURL.JoinPath(federatedUser.InboxPath).String(),
		Doer:     localUser,
		Payload:  payload,
	})
}
