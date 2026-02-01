// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgery

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

type Session struct {
	Client http.Client
	URL    url.URL
}

func (sess Session) Get(t testing.TB, uriParts ...string) Response {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), "GET", sess.URL.JoinPath(uriParts...).String(), nil)
	require.NoError(t, err)
	return Response{sess.Do(t, http.StatusOK, req)}
}

func (sess Session) Do(t testing.TB, expectedStatus int, req *http.Request) *http.Response {
	t.Helper()

	// allow URLs without host
	req.URL = sess.URL.ResolveReference(req.URL)

	resp, err := sess.Client.Do(req)
	require.NoError(t, err)
	if resp.StatusCode != expectedStatus {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s returned status %d, expected %d:\n%s", req.Method, req.URL, resp.StatusCode, expectedStatus, string(body))
		return nil
	}
	return resp
}
