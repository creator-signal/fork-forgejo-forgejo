// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"bytes"
	"fmt"
	"time"

	ap "github.com/go-ap/activitypub"
	json "github.com/go-ap/jsonld"
	"github.com/valyala/fastjson"
)

const (
	TicketType ap.ActivityVocabularyType = "Ticket"
)

// Ticket represents an item that requires work or attention.
type Ticket struct {
	ap.Object
	Fields TicketFields
}

// TicketFields represents the extension fields to the ActivityPub `Object` type.
type TicketFields struct {
	// Followers is a collection of the followers of the Ticket.
	Followers []Follower `jsonld:"followers,omitempty"`
	// Team is a collection of the project team members.
	Team []ProjectMember `jsonld:"team,omitempty"`
	// Dependants is a collection of Tickets that depend on this Ticket.
	Dependants []Dependant `jsonld:"dependants,omitempty"`
	// Dependants is a collection of Tickets on which this Ticket depends.
	Dependencies []Dependency `jsonld:"dependencies,omitempty"`
	// IsResolved indicates whether work on this Ticket is done.
	IsResolved bool `jsonld:"isResolved,omitempty"`
	// ResolvedBy represents who user or activity marked the Ticket resolved.
	ResolvedBy Resolver `jsonld:"resolvedBy,omitempty"`
	// Resolved represents when the Ticket was marked as resolved.
	Resolved time.Time `jsonld:"resolved,omitempty"`
	// Assignments represents the link to list of assigned actors.
	Assignments Assignments `jsonld:"assignments,omitempty"`
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

func ticketContext() []string {
	return []string{
		"https://www.w3.org/ns/activitystreams",
		"https://forgefed.org/ns",
	}
}

func ticketContextJSON() ([]byte, error) {
	ctx, err := json.Marshal(ticketContext())
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`"@context":%s`, string(ctx))), nil
}

// Context gets the canonical context strings for a Ticket.
func (t Ticket) TicketContext() []string {
	return ticketContext()
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	ctx, err := ticketContextJSON()
	if err != nil {
		return nil, err
	}

	obj, err := json.Marshal(t.Object)
	if err != nil {
		return nil, err
	}

	pre := fmt.Sprintf("{%s,%s", ctx, obj[1:len(obj)-1])
	res := bytes.NewBuffer([]byte(pre))

	if fields, err := json.Marshal(t.Fields); err != nil {
		return nil, err
	} else if _, err = res.Write([]byte(fmt.Sprintf(",%s", fields[1:len(fields)-1]))); err != nil {
		return nil, err
	}

	if err := res.WriteByte('}'); err != nil {
		return nil, err
	}

	return res.Bytes(), nil
}

func (t *Ticket) UnmarshalJSON(data []byte) error {
	tt, err := TicketUnmarshalJSON(data)
	if err == nil {
		*t = tt
	}

	return err
}

func TicketUnmarshalJSON(data []byte) (Ticket, error) {
	obj := ap.Object{}
	if err := obj.UnmarshalJSON(data); err != nil {
		return Ticket{}, err
	}

	ticket := NewTicket(obj)

	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return Ticket{}, err
	}

	i := 0
	for follower := val.GetObject("followers", fmt.Sprintf("%d", i)); follower != nil; i += 1 {
		f := Follower{}
		if err := f.UnmarshalJSON(follower.MarshalTo([]byte{})); err != nil {
			return Ticket{}, err
		}

		ticket.Fields.Followers = append(ticket.Fields.Followers, f)
	}

	i = 0
	for member := val.GetObject("team", fmt.Sprintf("%d", i)); member != nil; i += 1 {
		m := ProjectMember{}
		if err := m.UnmarshalJSON(member.MarshalTo([]byte{})); err != nil {
			return Ticket{}, err
		}

		ticket.Fields.Team = append(ticket.Fields.Team, m)
	}

	i = 0
	for dependant := val.GetObject("dependants", fmt.Sprintf("%d", i)); dependant != nil; i += 1 {
		d, err := TicketUnmarshalJSON(dependant.MarshalTo([]byte{}))
		if err != nil {
			return Ticket{}, err
		}

		ticket.Fields.Dependants = append(ticket.Fields.Dependants, Dependant(d))
	}

	i = 0
	for dependency := val.GetObject("dependencies", fmt.Sprintf("%d", i)); dependency != nil; i += 1 {
		d, err := TicketUnmarshalJSON(dependency.MarshalTo([]byte{}))
		if err != nil {
			return Ticket{}, err
		}

		ticket.Fields.Dependencies = append(ticket.Fields.Dependencies, Dependency(d))
	}

	ticket.Fields.IsResolved = val.GetBool("isResolved")

	if resolvedBy := val.GetObject("resolvedBy"); resolvedBy != nil {
		resActivity := Activity{}
		resActor := Actor{}
		if err := resActivity.UnmarshalJSON(resolvedBy.MarshalTo([]byte{})); err == nil {
			ticket.Fields.ResolvedBy = resActivity
		} else if err = resActor.UnmarshalJSON(resolvedBy.MarshalTo([]byte{})); err == nil {
			ticket.Fields.ResolvedBy = resActor
		} else {
			return Ticket{}, err
		}
	}

	if resolved := val.GetObject("resolved"); resolved != nil {
		if err := ticket.Fields.Resolved.UnmarshalJSON(resolved.MarshalTo([]byte{})); err != nil {
			return Ticket{}, err
		}
	}

	ticket.Fields.Assignments = ap.IRI(string(val.GetStringBytes("assignments")))

	return ticket, nil
}
