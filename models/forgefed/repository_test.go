// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RepositoryUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    Repository
		wantErr error
	}

	tests := map[string]testPair{
		"minimal repository": {
			item: []byte(`{
				"@context": [
					"https://www.w3.org/ns/activitystreams",
					"https://w3id.org/security/v2",
					"https://forgefed.org/ns"
				],
				"id": "https://dev.example/aviva/treesim",
				"type": "Repository",
				"publicKey": {
					"id": "https://dev.example/aviva/treesim#main-key",
					"owner": "https://dev.example/aviva/treesim",        
					"publicKeyPem": "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhki....."
				},
				"inbox": "https://dev.example/aviva/treesim/inbox",
				"outbox": "https://dev.example/aviva/treesim/outbox",
				"followers": "https://dev.example/aviva/treesim/followers",
				"name": "Tree Growth 3D Simulation",
				"summary": "<p>Tree growth 3D simulator for my nature exploration game</p>"}`),
			want: Repository{
				Object: ap.Object{
					ID:      ap.ID(ap.IRI("https://dev.example/aviva/treesim")),
					Type:    RepositoryType,
					Name:    ap.DefaultNaturalLanguageValue("Tree Growth 3D Simulation"),
					Summary: ap.DefaultNaturalLanguageValue("<p>Tree growth 3D simulator for my nature exploration game</p>"),
				},
				Fields: RepositoryFields{
					PublicKey: ap.PublicKey{
						ID:           ap.ID(ap.IRI("https://dev.example/aviva/treesim#main-key")),
						Owner:        ap.IRI("https://dev.example/aviva/treesim"),
						PublicKeyPem: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhki.....",
					},
					Inbox:     ap.IRI("https://dev.example/aviva/treesim/inbox"),
					Outbox:    ap.IRI("https://dev.example/aviva/treesim/outbox"),
					Followers: ap.IRI("https://dev.example/aviva/treesim/followers"),
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := RepositoryUnmarshalJSON(tt.item)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, got, tt.want, "UnmarshalJSON() got = %v, want %v", got, tt.want)
		})
	}
}

func Test_RepositoryMarshalJSON(t *testing.T) {
	type testPair struct {
		item    Repository
		wantErr error
	}

	tests := map[string]testPair{
		"minimal repository": {
			item: Repository{
				Object: ap.Object{
					ID:      ap.ID(ap.IRI("https://dev.example/aviva/treesim")),
					Type:    RepositoryType,
					Name:    ap.DefaultNaturalLanguageValue("Tree Growth 3D Simulation"),
					Summary: ap.DefaultNaturalLanguageValue("<p>Tree growth 3D simulator for my nature exploration game</p>"),
				},
				Fields: RepositoryFields{
					PublicKey: ap.PublicKey{
						ID:           ap.ID(ap.IRI("https://dev.example/aviva/treesim#main-key")),
						Owner:        ap.IRI("https://dev.example/aviva/treesim"),
						PublicKeyPem: "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhki.....",
					},
					Inbox:     ap.IRI("https://dev.example/aviva/treesim/inbox"),
					Outbox:    ap.IRI("https://dev.example/aviva/treesim/outbox"),
					Followers: ap.IRI("https://dev.example/aviva/treesim/followers"),
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.item.MarshalJSON()
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			gotItem, err := RepositoryUnmarshalJSON(got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.item, gotItem, "MarshalJSON() got = %v, want %v, encoded: %v", gotItem, tt.item, string(got))
		})
	}
}
