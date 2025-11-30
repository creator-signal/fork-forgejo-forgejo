package lock

import (
	"errors"
	"fmt"

	opentofu_model "forgejo.org/models/packages/opentofu"
	"forgejo.org/modules/json"
)

var ErrParseLockInfo = errors.New("failed to parse the lock info")

// ParseLockInfo extracts the lock information from a lock request body.
func ParseLockInfo(requestBody *[]byte) (*opentofu_model.StateLock, error) {
	var lockInfo opentofu_model.StateLock
	if err := json.Unmarshal(*requestBody, &lockInfo); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseLockInfo, err)
	}

	return &lockInfo, nil
}
