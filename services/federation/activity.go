package federation

import (
	"crypto/rand"
	"fmt"
	"strings"

	ap "github.com/go-ap/activitypub"
)

// GetActivityID is a helper function to extract an ActivityPub Activity ID.
//
// Extracts the ID text from the Activity ID IRI.
func GetActivityID(activity *ap.Activity) (string, error) {
	if activity == nil {
		return "", fmt.Errorf("nil Activity")
	}

	activityURL, err := activity.ID.URL()
	if err != nil {
		return "", err
	}

	parts := strings.Split(activityURL.Path, "/")
	pathLen := len(parts)
	boxIdx := pathLen - 2
	activityIDIdx := pathLen - 1

	// match the second-to-last path part: "../{inbox,outbox}/<activity-id>"
	if !(parts[boxIdx] == "inbox" || parts[boxIdx] == "outbox") {
		return "", fmt.Errorf("invalid ActivityPub outbox ID")
	}

	// get the last path part: "../{inbox,outbox}/<activity-id>"
	return parts[activityIDIdx], nil
}

// CreateActivityID creates a random 128-bit activity ID
func CreateActivityID() string {
	return rand.Text()
}
