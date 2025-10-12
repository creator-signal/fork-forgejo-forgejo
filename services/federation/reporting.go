// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
	"forgejo.org/models/moderation"
	"forgejo.org/models/repo"
	"forgejo.org/models/user"
	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"

	ap "github.com/go-ap/activitypub"
	"github.com/go-ap/jsonld"
	"github.com/google/uuid"
)

func BuildReport(reportUUID uuid.UUID, actorID string, reportedObjects []string, report *moderation.AbuseReport) ([]byte, *user.User, error) {
	reportedObjectsCollection := ap.CollectionNew(ap.NilID)
	err := reportedObjectsCollection.Append(ap.IRI(actorID))
	if err != nil {
		return nil, nil, err
	}

	for _, apID := range reportedObjects {
		err = reportedObjectsCollection.Append(ap.IRI(apID))
		if err != nil {
			return nil, nil, err
		}
	}

	apReportID := fmt.Sprintf("%sapi/v1/activitypub/reports/%s", setting.AppURL, reportUUID)
	apReport := ap.FlagNew(ap.ID(apReportID), reportedObjectsCollection.Items)

	instanceActor := user.NewAPServerActor()
	apReport.Actor = ap.ID(instanceActor.APActorID())
	err = apReport.Content.Append(
		ap.NilLangRef,
		fmt.Appendf([]byte{}, "%s: %s", report.Category.String(), report.Remarks),
	)
	if err != nil {
		return nil, nil, err
	}

	apReportJSON, err := jsonld.WithContext(jsonld.IRI(ap.ActivityBaseURI)).Marshal(apReport)
	if err != nil {
		return nil, nil, err
	}

	log.Debug("Built Flag action: %s", apReportJSON)

	return apReportJSON, instanceActor, nil
}

func ReportContent(
	ctx context.Context,
	report moderation.AbuseReport,
	apID string,
) (*uuid.UUID, error) {
	actorID := new(string)

	// Get the inbox of the actor the report is concerning
	actorInbox := ""
	federationHostID := int64(-1)

	switch report.ContentType {
	case moderation.ReportedContentTypeRepository:
		federatedRepo, err := repo.GetFollowingRepoByID(ctx, report.ContentID)
		if err != nil {
			return nil, err
		}

		// TODO: this is fragile. Save the original inbox URL and post to it
		actorInbox = fmt.Sprintf("%s/inbox", federatedRepo.URI)
		actorID = &federatedRepo.URI
		federationHostID = federatedRepo.FederationHostID
	case moderation.ReportedContentTypeUser:
		_, federatedUser, err := user.GetFederatedUserByUserID(ctx, report.ContentID)
		if err != nil {
			return nil, err
		}

		federationHost, err := forgefed.GetFederationHost(ctx, federatedUser.FederationHostID)
		if err != nil {
			return nil, err
		}

		federationHostURL := federationHost.AsURL()
		actorInbox = fmt.Sprintf("%s%s", federationHostURL.String(), federatedUser.InboxPath)
		actorID = &apID
		federationHostID = federationHost.ID
	case moderation.ReportedContentTypeComment, moderation.ReportedContentTypeIssue:
		// TODO: not federated yet
	}

	if actorInbox == "" || actorID == nil || federationHostID == -1 {
		return nil, fmt.Errorf("ActorInbox or ActorID is missing (is the respective implementation for %v missing?)", report.ContentType)
	}

	// Build "Flag" action
	// https://www.w3.org/TR/activitystreams-vocabulary/#dfn-flag
	// https://docs.joinmastodon.org/spec/activitypub/#Flag
	apReportUUID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	reportedObjects := []string{}
	if actorID != &apID && report.ContentType != moderation.ReportedContentTypeRepository {
		reportedObjects = append(reportedObjects, apID)
	}

	apReportJSON, instanceActor, err := BuildReport(apReportUUID, *actorID, reportedObjects, &report)
	if err != nil {
		return nil, err
	}

	IDString := ""
	for _, obj := range reportedObjects {
		IDString += obj
		IDString += ";"
	}

	activityPubIDs := sql.NullString{}
	if IDString != "" {
		activityPubIDs.String = strings.TrimSuffix(IDString, ";")
		activityPubIDs.Valid = true
	}

	forwardedReport := moderation.ForwardedAbuseReport{
		UUID:             apReportUUID.String(),
		FederationHostID: federationHostID,
		ActorID:          *actorID,
		ActivityPubIDs:   activityPubIDs,
	}

	_, err = db.GetEngine(ctx).Insert(forwardedReport)
	if err != nil {
		return nil, err
	}

	err = deliveryQueue.Push(deliveryQueueItem{
		Doer:          instanceActor,
		InboxURL:      actorInbox,
		Payload:       apReportJSON,
		DeliveryCount: 10,
	})
	if err != nil {
		return nil, err
	}

	return &apReportUUID, nil
}
