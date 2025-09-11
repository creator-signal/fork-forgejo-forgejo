// Copyright 2024, 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"

	"github.com/valyala/fastjson"
)

// Ticket represents an item that requires work or attention.
type Ticket struct {
	_context []string `json:"@context"`
	// Type represent the Object type, i.e. "Ticket".
	Type Type `json:"type"`
	// Id is the unique identifier for the Ticket.
	Id Id `json:"id"`
	// Context refers to the repo containing the relevant object.
	Context Context `json:"context"`
	// AttributedTo refers to the target of the Ticket.
	AttributedTo AttributedTo `json:"attributedTo"`
	// Summary is the header summary of the Ticket.
	Summary Summary `json:"summary"`
	// Content is the rendered body of the Ticket.
	Content Content `json:"content"`
	// MediaType is the HTTP media type for the rendered content, e.g. "text/html".
	MediaType MediaType `json:"mediaType"`
	// Source is the source content for the Ticket.
	Source TicketSource `json:"source"`
	// Published is the time that ticket submission was accepted.
	Published Published `json:"published"`
	// Followers is a collection of the followers of the Ticket.
	Followers []Follower `json:"followers"`
	// Team is a collection of the project team members.
	Team []ProjectMember `json:"team"`
	// Replies is a collection of direct comments on the Ticket.
	Replies []Reply `json:"replies"`
	// Dependants is a collection of Tickets that depend on this Ticket.
	Dependants []Dependant `json:"dependants"`
	// Dependants is a collection of Tickets on which this Ticket depends.
	Dependencies []Dependency `json:"dependencies"`
	// IsResolved indicates whether work on this Ticket is done.
	IsResolved bool `json:"isResolved"`
	// ResolvedBy represents who user or activity marked the Ticket resolved.
	ResolvedBy ResolvedBy `json:"resolvedBy"`
	// Resolved represents when the Ticket was marked as resolved.
	Resolved Resolved `json:"resolved"`
	// Assignments represents the link to list of assigned actors.
	Assignments Assignments `json:"assignments"`
}

type (
	Id = string
	// TODO: create an enum of valid `Type`s
	Type    = string
	Context = string
	// TODO: convert to a `User`/`Actor` type
	AttributedTo = string
	Summary      = string
	Content      = string
	// TODO: create an enum of valid media-type, maybe pull from net/http
	MediaType = string
	// TODO: change to a date-time type
	Published = string
	// TODO: convert to a proper `Follower` type
	Follower = string
	// TODO: convert to a proper `ProjectMember` type
	ProjectMember = string
	// TODO: convert to a proper `Reply` type
	Reply      = string
	Dependant  = Ticket
	Dependency = Ticket
	// TODO: convert to a `User`/`Actor` type
	ResolvedBy = string
	// TODO: change to a date-time type
	Resolved    = string
	Assignments = string
)

// NewTicket creates a minimally compliant `Ticket` instance.
func NewTicket(attributedTo AttributedTo, summary Summary, content Content) Ticket {
	return Ticket{
		_context: []string{
			"https://www.w3.org/ns/activitystreams",
			"https://forgefed.org/ns",
		},
		Type:         "Ticket",
		AttributedTo: attributedTo,
		Summary:      summary,
		Content:      content,
	}
}

// Context gets the canonical context strings for a Ticket.
func (t Ticket) ForgeFedContext() []string {
	return []string{
		"https://www.w3.org/ns/activitystreams",
		"https://forgefed.org/ns",
	}
}

// TODO: add parsing/validation of a valid Ticket
func TicketUnmarshalJSON(data []byte) (Ticket, error) {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return Ticket{}, err
	}

	attributedTo := AttributedTo(string(val.GetStringBytes("attributedTo")))
	summary := Summary(string(val.GetStringBytes("summary")))
	content := Content(string(val.GetStringBytes("content")))

	ticket := NewTicket(attributedTo, summary, content)

	ticket.Id = Id(string(val.GetStringBytes("id")))
	ticket.Context = Context(string(val.GetStringBytes("context")))
	ticket.MediaType = MediaType(string(val.GetStringBytes("mediaType")))

	ticket.Source.Content = Content(string(val.GetStringBytes("source", "content")))
	ticket.Source.MediaType = MediaType(string(val.GetStringBytes("source", "mediaType")))

	ticket.Published = Published(string(val.GetStringBytes("published")))

	i := 0
	follower := string(val.GetStringBytes("followers", fmt.Sprintf("%d", i)))
	for len(follower) != 0 {
		ticket.Followers = append(ticket.Followers, Follower(follower))

		i += 1
		follower = string(val.GetStringBytes("followers", fmt.Sprintf("%d", i)))
	}

	i = 0
	member := string(val.GetStringBytes("team", fmt.Sprintf("%d", i)))
	for len(member) != 0 {
		ticket.Team = append(ticket.Team, ProjectMember(member))

		i += 1
		member = string(val.GetStringBytes("team", fmt.Sprintf("%d", i)))
	}

	i = 0
	reply := string(val.GetStringBytes("replies", fmt.Sprintf("%d", i)))
	for len(reply) != 0 {
		ticket.Replies = append(ticket.Replies, Reply(reply))

		i += 1
		reply = string(val.GetStringBytes("replies", fmt.Sprintf("%d", i)))
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
	ticket.ResolvedBy = ResolvedBy(string(val.GetStringBytes("resolvedBy")))
	ticket.Resolved = Resolved(string(val.GetStringBytes("resolved")))
	ticket.Assignments = Assignments(string(val.GetStringBytes("assignments")))

	return ticket, nil
}

// TicketSource represents the source content text for a Ticket.
type TicketSource struct {
	// Content is the source text for the TicketSource.
	Content Content `json:"content"`
	// MediaType is the HTTP media type for the TicketSource, e.g. "text/markdown; variant=CommonMark"
	MediaType MediaType `json:"mediaType"`
}
