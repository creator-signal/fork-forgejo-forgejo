// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import "fmt"

type ErrNotAcceptable struct {
	Message string
}

func IsErrNotAcceptable(err error) bool {
	_, ok := err.(ErrNotAcceptable)
	return ok
}

func NewErrNotAcceptable(message string) ErrNotAcceptable {
	return ErrNotAcceptable{Message: message}
}

func (err ErrNotAcceptable) Error() string {
	return fmt.Sprintf("NotAcceptable: %v", err.Message)
}
