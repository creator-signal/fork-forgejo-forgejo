// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"bytes"
	"fmt"

	ap "github.com/go-ap/activitypub"
	json "github.com/go-ap/jsonld"
	"github.com/valyala/fastjson"
)

const (
	TicketTrackerType ap.ActivityVocabularyType = "TicketTracker"
)

// TicketTracker represents a project's ticket tracker.
//
// It follows the ForgeFed [TicketTracker](https://forgefed.org/spec/#tickettracker) specification.
type TicketTracker struct {
	ap.Object
	Fields TicketTrackerFields
}

// TicketTrackerFields extends the ActivityPub `Object` fields for the `TicketTracker` type.
type TicketTrackerFields struct {
	PublicKey ap.PublicKey `jsonld:"publicKey,omitempty"`
	Inbox     ap.IRI       `jsonld:"inbox,omitempty"`
	Outbox    ap.IRI       `jsonld:"outbox,omitempty"`
	Followers ap.IRI       `jsonld:"followers,omitempty"`
	Team      ap.IRI       `jsonld:"team,omitempty"`
}

// NewTicketTracker creates a minimally compliant `TicketTracker` instance.
func NewTicketTracker(obj ap.Object) TicketTracker {
	return TicketTracker{
		Object: obj,
	}
}

func ticketTrackerContext() []string {
	return []string{
		"https://www.w3.org/ns/activitystreams",
		"https://w3id.org/security/v2",
		"https://forgefed.org/ns",
	}
}

func ticketTrackerContextJSON() ([]byte, error) {
	ctx, err := json.Marshal(ticketContext())
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`"@context":%s`, string(ctx))), nil
}

func (t TicketTracker) MarshalJSON() ([]byte, error) {
	ctx, err := ticketTrackerContextJSON()
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

func (t *TicketTracker) UnmarshalJSON(data []byte) error {
	tt, err := TicketTrackerUnmarshalJSON(data)
	if err == nil {
		*t = tt
	}

	return err
}

func TicketTrackerUnmarshalJSON(data []byte) (TicketTracker, error) {
	obj := ap.Object{}
	if err := obj.UnmarshalJSON(data); err != nil {
		return TicketTracker{}, err
	}

	if obj.Type != TicketTrackerType {
		return TicketTracker{}, fmt.Errorf("invalid TicketTracker type, have: %v, got: %v", obj.Type, TicketTrackerType)
	}

	ticketTracker := NewTicketTracker(obj)

	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return TicketTracker{}, err
	}

	if publicKey := val.GetObject("publicKey"); publicKey == nil {
		return TicketTracker{}, fmt.Errorf("missing public key: %v", val)
	} else if err = ticketTracker.Fields.PublicKey.UnmarshalJSON(publicKey.MarshalTo([]byte{})); err != nil {
		return TicketTracker{}, err
	}

	ticketTracker.Fields.Inbox = ap.IRI(string(val.GetStringBytes("inbox")))
	ticketTracker.Fields.Outbox = ap.IRI(string(val.GetStringBytes("outbox")))
	ticketTracker.Fields.Followers = ap.IRI(string(val.GetStringBytes("followers")))
	ticketTracker.Fields.Team = ap.IRI(string(val.GetStringBytes("team")))

	return ticketTracker, nil
}
