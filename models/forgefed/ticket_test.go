// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgefed

import (
	"testing"

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
				_context: []string{
					"https://www.w3.org/ns/activitystreams",
					"https://forgefed.org/ns",
				},
				Type:         Type("Ticket"),
				Id:           Id("https://example.dev/alice/myrepo/issues/42"),
				Context:      Context("https://example.dev/alice/myrepo"),
				AttributedTo: AttributedTo("https://dev.community/bob"),
				Summary:      Summary("Nothing works!"),
				Content:      Content("<p>Please fix. <i>Everything</i> is broken!</p>"),
				MediaType:    MediaType("text/html"),
				Source: TicketSource{
					Content:   Content("Please fix. *Everything* is broken!"),
					MediaType: MediaType("text/markdown; variant=CommonMark"),
				},
				Assignments: Assignments("https://example.dev/alice/myrepo/issues/42/assignments"),
				IsResolved:  false,
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
