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
	RepositoryType ap.ActivityVocabularyType = "Repository"
)

// Repository represents a project's source-code repository.
//
// It follows the ForgeFed [Repository](https://forgefed.org/spec/#repository) specification.
type Repository struct {
	ap.Object
	Fields RepositoryFields
}

// RepositoryFields extends the ActivityPub `Object` fields for the `Repository` type.
type RepositoryFields struct {
	PublicKey ap.PublicKey `jsonld:"publicKey,omitempty"`
	Inbox     ap.IRI       `jsonld:"inbox,omitempty"`
	Outbox    ap.IRI       `jsonld:"outbox,omitempty"`
	Followers ap.IRI       `jsonld:"followers,omitempty"`
	Team      ap.IRI       `jsonld:"team,omitempty"`
}

// NewRepository creates a minimally compliant `Repository` instance.
func NewRepository(obj ap.Object) Repository {
	return Repository{
		Object: obj,
	}
}

func repositoryContext() []string {
	return []string{
		"https://www.w3.org/ns/activitystreams",
		"https://w3id.org/security/v2",
		"https://forgefed.org/ns",
	}
}

func repositoryContextJSON() ([]byte, error) {
	ctx, err := json.Marshal(ticketContext())
	if err != nil {
		return nil, err
	}

	return []byte(fmt.Sprintf(`"@context":%s`, string(ctx))), nil
}

func (t Repository) MarshalJSON() ([]byte, error) {
	ctx, err := repositoryContextJSON()
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

func (t *Repository) UnmarshalJSON(data []byte) error {
	tt, err := RepositoryUnmarshalJSON(data)
	if err == nil {
		*t = tt
	}

	return err
}

func RepositoryUnmarshalJSON(data []byte) (Repository, error) {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return Repository{}, err
	}

	obj := ap.Object{}
	if err := obj.UnmarshalJSON(data); err != nil {
		return Repository{}, err
	}

	if obj.Type != RepositoryType {
		return Repository{}, fmt.Errorf("invalid Repository type, have: %v, got: %v", obj.Type, RepositoryType)
	}

	repository := NewRepository(obj)

	if publicKey := val.GetObject("publicKey"); publicKey == nil {
		return Repository{}, fmt.Errorf("missing public key: %v", val)
	} else if err = repository.Fields.PublicKey.UnmarshalJSON(publicKey.MarshalTo([]byte{})); err != nil {
		return Repository{}, err
	}

	repository.Fields.Inbox = ap.IRI(string(val.GetStringBytes("inbox")))
	repository.Fields.Outbox = ap.IRI(string(val.GetStringBytes("outbox")))
	repository.Fields.Followers = ap.IRI(string(val.GetStringBytes("followers")))
	repository.Fields.Team = ap.IRI(string(val.GetStringBytes("team")))

	return repository, nil
}

// RepositoryActivity is used to store federated inbox/outbox activities.
type RepositoryActivity struct {
	ID int64 `xorm:"pk"`
	RepoID int64 `xorm:"repo_id"`
	ActivityID string `xorm:"TEXT 'activity_id'"`
	Activity string `xorm:"TEXT 'activity'"`
}
