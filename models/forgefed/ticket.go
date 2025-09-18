// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"time"

	ap "github.com/go-ap/activitypub"
	"github.com/valyala/fastjson"
)

// Ticket represents an item that requires work or attention.
type Ticket struct {
	ap.Object
	// Followers is a collection of the followers of the Ticket.
	Followers []Follower `json:"followers,omitempty"`
	// Team is a collection of the project team members.
	Team []ProjectMember `json:"team,omitempty"`
	// Dependants is a collection of Tickets that depend on this Ticket.
	Dependants []Dependant `json:"dependants,omitempty"`
	// Dependants is a collection of Tickets on which this Ticket depends.
	Dependencies []Dependency `json:"dependencies,omitempty"`
	// IsResolved indicates whether work on this Ticket is done.
	IsResolved bool `json:"isResolved,omitempty"`
	// ResolvedBy represents who user or activity marked the Ticket resolved.
	ResolvedBy Resolver `json:"resolvedBy,omitempty"`
	// Resolved represents when the Ticket was marked as resolved.
	Resolved time.Time `json:"resolved,omitempty"`
	// Assignments represents the link to list of assigned actors.
	Assignments Assignments `json:"assignments,omitempty"`
}

type (
	Follower      = ap.Actor
	ProjectMember = ap.Actor
	Dependant     = Ticket
	Dependency    = Ticket
	Assignments   = ap.Item
)

type Activity struct {
	ap.Activity
}

type Actor struct {
	ap.Actor
}

// NewTicket creates a minimally compliant `Ticket` instance.
func NewTicket(obj ap.Object) Ticket {
	return Ticket{
		Object: obj,
	}
}

func forgeFedContext() []string {
	return []string{
		"https://www.w3.org/ns/activitystreams",
		"https://forgefed.org/ns",
	}
}

// Context gets the canonical context strings for a Ticket.
func (t Ticket) ForgeFedContext() []string {
	return forgeFedContext()
}

// TODO: add parsing/validation of a valid Ticket
func TicketUnmarshalJSON(data []byte) (Ticket, error) {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return Ticket{}, err
	}

	obj := ap.Object{}
	if err := obj.UnmarshalJSON(data); err != nil {
		return Ticket{}, err
	}

	ticket := NewTicket(obj)

	ticket.ID = ap.ID(ap.IRI(string(val.GetStringBytes("id"))))

	i := 0
	follower := val.GetStringBytes("followers", fmt.Sprintf("%d", i))
	for len(follower) != 0 {
		f := Follower{}
		if err := f.UnmarshalJSON(follower); err != nil {
			return Ticket{}, err
		}

		ticket.Followers = append(ticket.Followers, f)

		i += 1
		follower = val.GetStringBytes("followers", fmt.Sprintf("%d", i))
	}

	i = 0
	member := val.GetStringBytes("team", fmt.Sprintf("%d", i))
	for len(member) != 0 {
		m := ProjectMember{}
		if err := m.UnmarshalJSON(member); err != nil {
			return Ticket{}, err
		}

		ticket.Team = append(ticket.Team, m)

		i += 1
		member = val.GetStringBytes("team", fmt.Sprintf("%d", i))
	}

	i = 0
	dependant := val.GetStringBytes("dependants", fmt.Sprintf("%d", i))
	for len(dependant) != 0 {
		d, err := TicketUnmarshalJSON(dependant)
		if err != nil {
			return Ticket{}, err
		}

		ticket.Dependants = append(ticket.Dependants, Dependant(d))

		i += 1
		dependant = val.GetStringBytes("dependants", fmt.Sprintf("%d", i))
	}

	i = 0
	dependency := val.GetStringBytes("dependencies", fmt.Sprintf("%d", i))
	for len(dependency) != 0 {
		d, err := TicketUnmarshalJSON(dependency)
		if err != nil {
			return Ticket{}, err
		}

		ticket.Dependencies = append(ticket.Dependencies, Dependency(d))

		i += 1
		dependency = val.GetStringBytes("dependencies", fmt.Sprintf("%d", i))
	}

	ticket.IsResolved = val.GetBool("isResolved")

	resolvedBy := val.GetStringBytes("resolvedBy")
	if len(resolvedBy) > 0 {
		resActivity := Activity{}
		resActor := Actor{}
		if err := resActivity.UnmarshalJSON(resolvedBy); err == nil {
			ticket.ResolvedBy = resActivity
		} else if err = resActor.UnmarshalJSON(resolvedBy); err == nil {
			ticket.ResolvedBy = resActor
		} else {
			return Ticket{}, err
		}
	}

	resolvedJSON := val.GetStringBytes("resolved")
	if len(resolvedJSON) > 0 {
		resolved := time.Time{}
		if err := resolved.UnmarshalJSON(resolvedJSON); err != nil {
			return Ticket{}, err
		}

		ticket.Resolved = resolved
	}

	ticket.Assignments = ap.IRI(string(val.GetStringBytes("assignments")))

	return ticket, nil
}
