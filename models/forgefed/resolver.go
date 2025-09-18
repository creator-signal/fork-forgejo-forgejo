// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"

	ap "github.com/go-ap/activitypub"
)

// Resolver is an interface for the `Ticket.ResolvedBy` field.
//
// The `Ticket.ResolvedBy` field can be either an ActivityPub `Activity` or `Actor`.
type Resolver interface {
	IsActivity() bool
	IsActor() bool
	ToActivity() (*ap.Activity, error)
	ToActor() (*ap.Actor, error)
}

func (a Activity) IsActivity() bool {
	return true
}

func (a Activity) IsActor() bool {
	return false
}

func (a Activity) ToActivity() (*ap.Activity, error) {
	return &a.Activity, nil
}

func (a Activity) ToActor() (*ap.Actor, error) {
	return nil, fmt.Errorf("Resolver is not an ActivityPub Actor")
}

func (a Actor) IsActivity() bool {
	return true
}

func (a Actor) IsActor() bool {
	return false
}

func (a Actor) ToActivity() (*ap.Activity, error) {
	return nil, fmt.Errorf("Resolver is not an ActivityPub Activity")
}

func (a Actor) ToActor() (*ap.Actor, error) {
	return &a.Actor, nil
}
