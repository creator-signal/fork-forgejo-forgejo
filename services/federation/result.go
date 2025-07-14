// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package federation

import "github.com/go-ap/activitypub"

type ServiceResult struct {
	HttpStatus   int
	Bytes        []byte
	Activity     activitypub.Activity
	withBytes    bool
	withActivity bool
	statusOnly   bool
}

func NewServiceResultStatusOnly(status int) ServiceResult {
	return ServiceResult{HttpStatus: status, statusOnly: true}
}

func NewServiceResultWithBytes(status int, bytes []byte) ServiceResult {
	return ServiceResult{HttpStatus: status, Bytes: bytes, withBytes: true}
}

func (self ServiceResult) WithBytes() bool {
	return self.withBytes
}

func (self ServiceResult) WithActivity() bool {
	return self.withActivity
}

func (self ServiceResult) StatusOnly() bool {
	return self.statusOnly
}
