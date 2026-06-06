// Copyright 2025 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"encoding/xml"
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedIssue(t *testing.T) {
	t.Run("RSS", func(t *testing.T) {
		defer tests.PrepareTestEnv(t)()

		req := NewRequest(t, "GET", "/user2/repo1/issues/1.rss")
		resp := MakeRequest(t, req, http.StatusOK)

		data := resp.Body.String()
		assert.Contains(t, data, `<rss version="2.0"`)

		var rss RSS
		err := xml.Unmarshal(resp.Body.Bytes(), &rss)
		require.NoError(t, err)
		assert.Contains(t, rss.Channel.Link, "/user2/repo1/issues/1")
		assert.NotEmpty(t, rss.Channel.PubDate)
		assert.Len(t, rss.Channel.Items, 40)
		assert.Equal(t, "good work!", rss.Channel.Items[1].Description)
		assert.NotEmpty(t, rss.Channel.Items[1].PubDate)
	})
}
