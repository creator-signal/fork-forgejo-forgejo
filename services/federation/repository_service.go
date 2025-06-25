// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"fmt"
	"net/http"

	// TODO: remove unneeded imports in comments
	// "forgejo.org/models/user"
	// "forgejo.org/modules/forgefed"
	"forgejo.org/modules/log"
	context_service "forgejo.org/services/context"

	ap "github.com/go-ap/activitypub"
	// "github.com/go-ap/jsonld"
)

func ProcessRepositoryInbox(ctx *context_service.APIContext, form any) {
	activity := form.(*ap.Activity)

	switch activity.Type {
	case ap.LikeType:
		repository := ctx.Repo.Repository
		likeActivity, s, err := ProcessLikeActivity(ctx, activity, repository.ID)
		if err != nil {

			return
		}
		return
	case ap.UndoType:
		processRepositoryInboxUndo(ctx, activity)
		return
	case ap.AcceptType:
		processRepositoryInboxAccept(ctx, activity)
		return
	}

	log.Error("Unsupported PersonInbox activity: %v", activity.Type)
	ctx.Error(http.StatusNotAcceptable, "Unsupported acvitiy", fmt.Errorf("Unsupported activity: %v", activity.Type))
}

// TODO:
// 1. remove FollowRemoteActor, no similar needage for repository, am I right?
// 2. sth. else to add?
/*
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
*/
