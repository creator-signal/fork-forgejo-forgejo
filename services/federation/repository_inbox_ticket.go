package federation

import (
	"context"
	"fmt"
	"net/http"

	db "forgejo.org/models/db"
	forgefed "forgejo.org/models/forgefed"
	ap "github.com/go-ap/activitypub"
)

// ProcessInboxTicketActivity handles a ForgeFed ticket request for a repository inbox.
//
// Allows federated peers to submit ticket information for a repository.
func ProcessInboxTicketActivity(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	// TODO: check the requesting Actor has the `report` Role for this repository

	activityID, err := GetActivityID(activity)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusBadRequest), err
	}

	// TODO: use ticket to open an issue on the repository
	//ticket := activity.Object.(*forgefed.Ticket)

	ctx, committer, err := db.TxContext(ctx)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}
	defer committer.Close()

	offerJSON, err := activity.MarshalJSON()
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	dbActivity := forgefed.RepositoryActivity{
		ID:         repositoryID,
		ActivityID: activityID,
		Activity:   string(offerJSON),
	}

	// add the ticket to the repository
	err = db.Insert(ctx, dbActivity)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	acceptActivityID := CreateActivityID()
	if len(activity.To) == 0 {
		return NewServiceResultStatusOnly(http.StatusBadRequest), err
	}
	acceptTo := activity.To[0].GetLink()
	acceptActivityIRI := ap.ID(fmt.Sprintf("%v/outbox/%v", acceptTo, repositoryID, acceptActivityID))

	acceptActivity := ap.AcceptNew(acceptActivityIRI, ap.Item(activity.ID))
	acceptJSON, err := acceptActivity.MarshalJSON()
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	dbAccept := forgefed.RepositoryActivity{
		RepoID:     repositoryID,
		ActivityID: acceptActivityID,
		Activity:   string(acceptJSON),
	}

	// add the Accept activity to the database
	err = db.Insert(ctx, dbAccept)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	err = committer.Commit()
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	return NewServiceResultWithBytes(http.StatusOK, acceptJSON), nil
}
