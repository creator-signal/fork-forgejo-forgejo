package federation

import (
	"context"
	"net/http"

	ap "github.com/go-ap/activitypub"

	"forgejo.org/models/db"
	"forgejo.org/models/forgefed"
)

// ProcessOutboxActivity handles a ForgeFed request for a repository outbox.
//
// Allows federated peers to retrieve outbox activity information for a repository.
func ProcessOutboxActivity(ctx context.Context, activity *ap.Activity, repositoryID int64) (ServiceResult, error) {
	// TODO: check ForgeFed Grant for appropriate permissions

	// get the last path part: "../outbox/<activity-id>"
	activityID, err := GetActivityID(activity)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusBadRequest), err
	}

	var dbActivity forgefed.RepositoryActivity

	err = db.GetEngine(ctx).Where("repo_id=? AND activity_id=?", repositoryID, activityID).Find(&dbActivity)
	if err != nil {
		return NewServiceResultStatusOnly(http.StatusInternalServerError), err
	}

	return NewServiceResultWithBytes(http.StatusOK, []byte(dbActivity.Activity)), nil
}
