package lock

import (
	"testing"
	"time"

	opentofu_model "forgejo.org/models/packages/opentofu"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	lockRequestPayload string = `{
		"ID": "55769533-286a-96d9-eb47-935c08f675a9",
		"Operation": "OperationTypeApply",
		"Info": "",
		"Who": "user@laptop",
		"Version": "1.10.6",
		"Created": "2025-12-01T16:37:39.015107776Z",
		"Path": ""
	}`
)

func TestParseLockInfo(t *testing.T) {
	t.Run("InvalidJSON", func(t *testing.T) {
		lockRequest := []byte("This is not a JSON file")

		lockInfo, err := ParseLockInfo(&lockRequest)
		assert.Nil(t, lockInfo)
		require.ErrorIs(t, err, ErrParseLockInfo)
	})

	t.Run("ValidLockRequestPayload", func(t *testing.T) {
		lockRequest := []byte(lockRequestPayload)

		createdTimestamp, _ := time.Parse(time.RFC3339, "2025-12-01T16:37:39.015107776Z")

		lockInfo, err := ParseLockInfo(&lockRequest)
		assert.Equal(t, opentofu_model.StateLock{
			LockID:        "55769533-286a-96d9-eb47-935c08f675a9",
			Operation:     "OperationTypeApply",
			UserName:      "user@laptop",
			ClientVersion: "1.10.6",
			CreatedUnix:   createdTimestamp,
			Path:          "",
		}, *lockInfo)
		assert.NoError(t, err)
	})
}
