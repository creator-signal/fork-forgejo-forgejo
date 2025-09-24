// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

	ap "github.com/go-ap/activitypub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_TicketUnmarshalJSON(t *testing.T) {
	type testPair struct {
		item    []byte
		want    Ticket
		wantErr error
	}

	tests := map[string]testPair{
		"minimal ticket": {
			item: []byte(`
			{
				"@context": [
					"https://www.w3.org/ns/activitystreams",
					"https://forgefed.org/ns"
				],
				"type": "Ticket",
				"id": "https://example.dev/alice/myrepo/issues/42",
				"context": "https://example.dev/alice/myrepo",
				"attributedTo": "https://dev.community/bob",
				"summary": "Nothing works!",
				"content": "<p>Please fix. <i>Everything</i> is broken!</p>",
				"mediaType": "text/html",
				"source": {
					"content": "Please fix. *Everything* is broken!",
					"mediaType": "text/markdown; variant=CommonMark"
				},
				"assignments": "https://example.dev/alice/myrepo/issues/42/assignments",
				"isResolved": false
			}`),
			want: Ticket{
				Object: ap.Object{
					ID:           ap.ID(ap.IRI("https://example.dev/alice/myrepo/issues/42")),
					Type:         TicketType,
					Context:      ap.IRI("https://example.dev/alice/myrepo"),
					AttributedTo: ap.IRI("https://dev.community/bob"),
					Summary:      ap.DefaultNaturalLanguageValue("Nothing works!"),
					Content:      ap.DefaultNaturalLanguageValue("<p>Please fix. <i>Everything</i> is broken!</p>"),
					MediaType:    ap.MimeType("text/html"),
					Source: ap.Source{
						Content:   ap.DefaultNaturalLanguageValue("Please fix. *Everything* is broken!"),
						MediaType: ap.MimeType("text/markdown; variant=CommonMark"),
					},
				},
				Fields: TicketFields{
					Assignments: ap.IRI("https://example.dev/alice/myrepo/issues/42/assignments"),
					IsResolved:  false,
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := TicketUnmarshalJSON(tt.item)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, got, tt.want, "UnmarshalJSON() got = %v, want %v", got, tt.want)
		})
	}
}

func Test_TicketMarshalJSON(t *testing.T) {
	type testPair struct {
		item    Ticket
		wantErr error
	}

	tests := map[string]testPair{
		"minimal ticket": {
			item: Ticket{
				Object: ap.Object{
					ID:           ap.ID(ap.IRI("https://example.dev/alice/myrepo/issues/42")),
					Type:         TicketType,
					Context:      ap.IRI("https://example.dev/alice/myrepo"),
					AttributedTo: ap.IRI("https://dev.community/bob"),
					Summary:      ap.DefaultNaturalLanguageValue("Nothing works!"),
					Content:      ap.DefaultNaturalLanguageValue("<p>Please fix. <i>Everything</i> is broken!</p>"),
					MediaType:    ap.MimeType("text/html"),
					Source: ap.Source{
						Content:   ap.DefaultNaturalLanguageValue("Please fix. *Everything* is broken!"),
						MediaType: ap.MimeType("text/markdown; variant=CommonMark"),
					},
				},
				Fields: TicketFields{
					Assignments: ap.IRI("https://example.dev/alice/myrepo/issues/42/assignments"),
					IsResolved:  false,
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

			gotItem, err := TicketUnmarshalJSON(got)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.item, gotItem, "MarshalJSON() got = %v, want %v, encoded: %v", gotItem, tt.item, string(got))
		})
	}
}
