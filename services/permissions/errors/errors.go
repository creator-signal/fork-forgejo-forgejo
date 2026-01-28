// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package errors

import (
	"fmt"
)

func NewError(err permissionsError, message string, args ...any) error {
	return fmt.Errorf("%w: "+message, append([]any{err}, args...)...)
}

type permissionsError string

func (err permissionsError) Error() string { return string(err) }

var NotFound = permissionsError("NotFound")

func NewNotFound(message string, args ...any) error {
	return NewError(NotFound, message, args...)
}

var Server = permissionsError("Server")

func NewServer(message string, args ...any) error {
	return NewError(Server, message, args...)
}

var Forbidden = permissionsError("Forbidden")

func NewForbidden(message string, args ...any) error {
	return NewError(Forbidden, message, args...)
}
