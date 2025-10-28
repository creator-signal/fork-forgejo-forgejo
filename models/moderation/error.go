// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package moderation

import (
	"fmt"

	"github.com/google/uuid"
)

type ErrReportNotExist struct {
	id string
}

func (err ErrReportNotExist) Error() string {
	return fmt.Sprintf("report does not exists [id: %s]", err.id)
}

func IsErrReportNotExists(err error) bool {
	_, ok := err.(ErrReportNotExist)
	return ok
}

type ErrForwardedReportNotExist struct {
	uuid uuid.UUID
}

func (err ErrForwardedReportNotExist) Error() string {
	return fmt.Sprintf("forwarded report does not exist [uuid: %s]", err.uuid)
}

func IsErrForwardedReportNotExists(err error) bool {
	_, ok := err.(ErrForwardedReportNotExist)
	return ok
}
