// Copyright 2023 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed_test

import (
	"fmt"
	"reflect"
	"testing"

	"forgejo.org/modules/forgefed"
	"forgejo.org/modules/json"
	"forgejo.org/modules/util"
	"forgejo.org/modules/validation"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/require"
)

func Test_RepositoryMarshalJSON(t *testing.T) {
	type testPair struct {
		item    forgefed.Repository
		want    []byte
		wantErr error
	}

	tests := map[string]testPair{
		"empty": {
			item: forgefed.Repository{},
			want: nil,
		},
		"with ID": {
			item: forgefed.Repository{
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
				Team: nil,
			},
			want: []byte(`{"id":"https://example.com/1"}`),
		},
		"with Team as IRI": {
			item: forgefed.Repository{
				Team: ap.IRI("https://example.com/1"),
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":"https://example.com/1"}`),
		},
		"with Team as IRIs": {
			item: forgefed.Repository{
				Team: ap.ItemCollection{
					ap.IRI("https://example.com/1"),
					ap.IRI("https://example.com/2"),
				},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":["https://example.com/1","https://example.com/2"]}`),
		},
		"with Team as Object": {
			item: forgefed.Repository{
				Team: ap.Object{ID: "https://example.com/1"},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":{"id":"https://example.com/1"}}`),
		},
		"with Team as slice of Objects": {
			item: forgefed.Repository{
				Team: ap.ItemCollection{
					ap.Object{ID: "https://example.com/1"},
					ap.Object{ID: "https://example.com/2"},
				},
				Actor: ap.Actor{
					ID: "https://example.com/1",
				},
			},
			want: []byte(`{"id":"https://example.com/1","team":[{"id":"https://example.com/1"},{"id":"https://example.com/2"}]}`),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if (err != nil || tt.wantErr != nil) && tt.wantErr.Error() != err.Error() {
				t.Errorf("MarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %q, want %q", got, tt.want)
			}
		})
	}
}

func Test_RepositoryUnmarshalJSON(t *testing.T) {
	type testPair struct {
		data    []byte
		want    *forgefed.Repository
		wantErr error
	}

	tests := map[string]testPair{
		"nil": {
			data:    nil,
			wantErr: fmt.Errorf("cannot parse JSON: %w", fmt.Errorf("cannot parse empty string; unparsed tail: %q", "")),
		},
		"empty": {
			data:    []byte{},
			wantErr: fmt.Errorf("cannot parse JSON: %w", fmt.Errorf("cannot parse empty string; unparsed tail: %q", "")),
		},
		"with Type": {
			data: []byte(`{"type":"Repository"}`),
			want: &forgefed.Repository{
				Actor: ap.Actor{
					Type: forgefed.RepositoryType,
				},
			},
		},
		"with Type and ID": {
			data: []byte(`{"id":"https://example.com/1","type":"Repository"}`),
			want: &forgefed.Repository{
				Actor: ap.Actor{
					ID:   "https://example.com/1",
					Type: forgefed.RepositoryType,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := new(forgefed.Repository)
			err := got.UnmarshalJSON(tt.data)
			if tt.wantErr != nil && err == nil {
				t.Errorf("Expected UnmarshalJSON() wantErr = \"%v\"", tt.wantErr)
				return
			} else if tt.wantErr == nil && err != nil {
				t.Errorf("Unxpected UnmarshalJSON() error = \"%v\"", err)
				return
			} else if err != nil && tt.wantErr != nil && tt.wantErr.Error() != err.Error() {
				t.Errorf("UnmarshalJSON() error = \"%v\", wantErr \"%v\"", err, tt.wantErr)
				return
			}
			if tt.want != nil && !reflect.DeepEqual(got, tt.want) {
				jGot, _ := json.Marshal(got)
				jWant, _ := json.Marshal(tt.want)
				t.Errorf("UnmarshalJSON() got = %s, want %s", jGot, jWant)
			}
		})
	}
}

func Test_RepositoryValidation(t *testing.T) {
	objectID := "https://forgejo.org/user/repo"
	inbox := ap.IRI(objectID + "/inbox")
	outbox := ap.IRI(objectID + "/outbox")
	followers := ap.IRI(objectID + "/followers")
	team := ap.IRI(objectID + "/team")

	_, key, err := util.GenerateKeyPair(3072)
	require.NoError(t, err)

	publicKey := ap.PublicKey{
		ID:           ap.ID(fmt.Sprintf("%s#main-key", objectID)),
		Owner:        ap.IRI(objectID),
		PublicKeyPem: key,
	}

	sut := forgefed.Repository{
		Actor: ap.Actor{
			ID:        ap.ID(objectID),
			Type:      forgefed.RepositoryType,
			Name:      ap.DefaultNaturalLanguage("Test Repository"),
			Summary:   ap.DefaultNaturalLanguage("<p>A repository for ActivityPub test.</p>"),
			Inbox:     inbox,
			Outbox:    outbox,
			Followers: followers,
			PublicKey: publicKey,
		},
		Team: team,
	}

	_, err = validation.IsValid(sut)
	require.NoError(t, err, "expected valid Repository: %v", err)

	badIRI := ap.IRI("https://bad.url/%^*")

	sut.Actor.Inbox = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid Repository inbox: %v", sut)

	sut.Actor.Inbox = inbox
	sut.Actor.Outbox = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid Repository outbox: %v", sut)

	sut.Actor.Outbox = outbox
	sut.Actor.Followers = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid Repository followers: %v", sut)

	sut.Actor.Followers = followers
	sut.Team = badIRI

	_, err = validation.IsValid(sut)
	require.Error(t, err, "expected invalid Repository team: %v", sut)
}
