// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	forgefed_model "forgejo.org/models/forgefed"
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/valyala/fastjson"
)

const (
	RepositoryType ap.ActivityVocabularyType = "Repository"
)

type Repository struct {
	ap.Actor
	// Team Collection of actors who have management/push access to the repository
	Team ap.Item `jsonld:"team,omitempty"`
	// Forks OrderedCollection of repositories that are forks of this repository
	Forks ap.Item `jsonld:"forks,omitempty"`
	// ForkedFrom Identifies the repository which this repository was created as a fork
	ForkedFrom ap.Item `jsonld:"forkedFrom,omitempty"`
}

// RepositoryNew initializes a Repository type actor
func RepositoryNew(id ap.ID) *Repository {
	a := ap.ActorNew(id, RepositoryType)
	a.Type = RepositoryType
	o := Repository{Actor: *a}
	return &o
}

func (repo Repository) MarshalJSON() ([]byte, error) {
	b, err := repo.Actor.MarshalJSON()
	if len(b) == 0 || err != nil {
		return nil, err
	}

	b = b[:len(b)-1]
	if repo.Team != nil {
		ap.JSONWriteItemProp(&b, "team", repo.Team)
	}
	if repo.Forks != nil {
		ap.JSONWriteItemProp(&b, "forks", repo.Forks)
	}
	if repo.ForkedFrom != nil {
		ap.JSONWriteItemProp(&b, "forkedFrom", repo.ForkedFrom)
	}
	ap.JSONWrite(&b, '}')
	return b, nil
}

func JSONLoadRepository(val *fastjson.Value, r *Repository) error {
	if err := ap.OnActor(&r.Actor, func(a *ap.Actor) error {
		return ap.JSONLoadActor(val, a)
	}); err != nil {
		return err
	}

	r.Team = ap.JSONGetItem(val, "team")
	r.Forks = ap.JSONGetItem(val, "forks")
	r.ForkedFrom = ap.JSONGetItem(val, "forkedFrom")

	_, err := validation.IsValid(r)

	return err
}

func (repo *Repository) UnmarshalJSON(data []byte) error {
	p := fastjson.Parser{}
	val, err := p.ParseBytes(data)
	if err != nil {
		return err
	}
	return JSONLoadRepository(val, repo)
}

// ToRepository tries to convert the it Item to a Repository Actor.
func ToRepository(it ap.Item) (*Repository, error) {
	switch i := it.(type) {
	case *Repository:
		return i, nil
	case Repository:
		return &i, nil
	case *ap.Actor:
		return (*Repository)(unsafe.Pointer(i)), nil
	case ap.Actor:
		return (*Repository)(unsafe.Pointer(&i)), nil
	default:
		// NOTE(marius): this is an ugly way of dealing with the interface conversion error: types from different scopes
		typ := reflect.TypeOf(new(Repository))
		if i, ok := reflect.ValueOf(it).Convert(typ).Interface().(*Repository); ok {
			return i, nil
		}
	}
	return nil, ap.ErrorInvalidType[ap.Actor](it)
}

type withRepositoryFn func(*Repository) error

// OnRepository calls function fn on it Item if it can be asserted to type *Repository
func OnRepository(it ap.Item, fn withRepositoryFn) error {
	if it == nil {
		return nil
	}
	ob, err := ToRepository(it)
	if err != nil {
		return err
	}
	return fn(ob)
}

// OwnerID derives the owner (user, org, etc) from the ActivityPub ID.
func (repo Repository) OwnerID() (ap.IRI, error) {
	idURL, err := repo.ID.URL()
	if err != nil {
		return ap.IRI(""), err
	}

	// This is somewhat fragile for implementations that do not use the pattern:
	// - forge.url/owner/repo
	// - forge.url/org/owner/repo
	//
	// However, Forgejo does follow this pattern, as well as most (all?) other popular forges
	idPathParts := strings.Split(strings.Trim(idURL.Path, "/"), "/")
	pathLen := len(idPathParts)
	if pathLen < 2 {
		return ap.IRI(""), fmt.Errorf("invalid repository ID: %s", idURL.String())
	}

	ownerPath := strings.Join(idPathParts[:pathLen-1], "/")
	idURL.Path = "/" + ownerPath

	return ap.IRI(idURL.String()), nil
}

// FromActivityPubRepository attempts to convert an ActivityPub `Repository` object into a `FederatedRepository` databse record.
//
// `ownerID` is the database ID for the owning `FederatedUser`. It can be fetched using code similar to:
//
//	repo := forgefed.Repository{ ... }
//	ownerIRI, _err := repo.OwnerID()
//	_, federatedUser, _err := user.FindFederatedUserByExternalID(context.Background(), ownerIRI)
//	ownerID := federatedUser.ID
func (repo Repository) IntoFederatedRepository(ownerID int64) (*forgefed_model.FederatedRepository, error) {
	if ownerID < 0 {
		return nil, fmt.Errorf("invalid owner ID: %d", ownerID)
	}
	if _, err := repo.OwnerID(); err != nil {
		return nil, err
	}

	res := forgefed_model.FederatedRepository{
		OwnerID:  ownerID,
		ObjectID: repo.ID,
	}

	if repo.Name != nil {
		res.Name = repo.Name.String()
	}
	if repo.Summary != nil {
		res.Summary = repo.Summary.String()
	}
	if repo.Inbox != nil && repo.Inbox.IsLink() {
		res.Inbox = repo.Inbox.GetLink()
	}
	if repo.Outbox != nil && repo.Outbox.IsLink() {
		res.Outbox = repo.Outbox.GetLink()
	}
	if repo.Followers != nil && repo.Followers.IsLink() {
		res.Followers = repo.Followers.GetLink()
	}
	if repo.Team != nil && repo.Team.IsLink() {
		res.Team = repo.Team.GetLink()
	}

	return &res, nil
}

// Validate performs checks to validate the `Repository`.
func (repo Repository) Validate() []string {
	var (
		res     []string
		emptyID ap.ID
	)

	if repo.ID != emptyID {
		res = append(res, validation.ValidateIRI(repo.ID, "ID")...)
	}

	res = append(res, repo.validateObjectType()...)
	if repo.Inbox != nil && repo.Inbox.IsLink() {
		res = append(res, validation.ValidateIRI(repo.Inbox.GetLink(), "Inbox")...)
	}
	if repo.Outbox != nil && repo.Outbox.IsLink() {
		res = append(res, validation.ValidateIRI(repo.Outbox.GetLink(), "Outbox")...)
	}
	if repo.Followers != nil && repo.Followers.IsLink() {
		res = append(res, validation.ValidateIRI(repo.Followers.GetLink(), "Followers")...)
	}
	if repo.Team != nil && repo.Team.IsLink() {
		res = append(res, validation.ValidateIRI(repo.Team.GetLink(), "Team")...)
	}
	res = append(res, repo.validatePublicKey()...)

	return res
}

// validateObjectType checks that the ActivityPub Object Type matches the expected value.
func (repo Repository) validateObjectType() []string {
	var res []string

	if repo.Type != RepositoryType {
		res = []string{fmt.Sprintf("invalid object type: %v, expected: %v", repo.Type, RepositoryType)}
	}

	return res
}

// validatePublicKey checks that the ActivityPub Public Key object is valid.
func (repo Repository) validatePublicKey() []string {
	var (
		res      []string
		emptyKey ap.PublicKey
	)

	if repo.PublicKey != emptyKey {
		res = append(res, validation.ValidateIRI(repo.PublicKey.ID, "PublicKey.ID")...)
		res = append(res, validation.ValidateIRI(repo.PublicKey.Owner, "PublicKey.Owner")...)
		res = append(res, validation.ValidatePublicKey([]byte(repo.PublicKey.PublicKeyPem), "PublicKey.PublicKeyPem")...)

		if repo.ID != repo.PublicKey.Owner {
			res = append(res, fmt.Sprintf("invalid public key owner, have: %s, expected: %s", repo.PublicKey.Owner, repo.ID))
		}
	}

	return res
}
