// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package webfinger

import "fmt"

type Malformed struct {
	extraInfo string
}

func (err Malformed) Error() string {
	return fmt.Sprintf("Malformed webfinger object: %s", err.extraInfo)
}

func IsWebfingerMalformed(err error) bool {
	_, ok := err.(Malformed)
	return ok
}

type MissingActivityEndpoint struct{}

func (MissingActivityEndpoint) Error() string {
	return "Webfinger JRD is missing an application/acitvity+json field"
}

func IsMissingActivityEndpoint(err error) bool {
	_, ok := err.(MissingActivityEndpoint)
	return ok
}
