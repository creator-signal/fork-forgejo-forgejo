// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
)

// ErrFederatedRepoNotExist represents an error for no `FederatedRepository` entry found in the database.
type ErrFederatedRepoNotExist struct {
	Name string
}

// IsErrFederatedRepoNotExist checks if the error is an `ErrFederatedRepoNotExist` error.
func IsErrFederatedRepoNotExist(err error) bool {
	_, ok := err.(ErrFederatedRepoNotExist)
	return ok
}

func (err ErrFederatedRepoNotExist) Error() string {
	return fmt.Sprintf("federated repository does not exist [name: %s]", err.Name)
}
