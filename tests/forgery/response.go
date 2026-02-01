// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package forgery

import (
	"io"
	"net/http"
	"testing"

	"forgejo.org/modules/json"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/require"
)

type Response struct {
	HTTP *http.Response
}

// Read implements io.Reader.
func (r Response) Read(p []byte) (n int, err error) {
	return r.HTTP.Body.Read(p)
}

// HTMLDoc returns a parsed HTML document from the response.
func (r Response) HTMLDoc(t testing.TB) HTMLDoc {
	t.Helper()
	doc, err := goquery.NewDocumentFromReader(r)
	require.NoError(t, err)
	return HTMLDoc{doc}
}

type HTMLDoc struct {
	*goquery.Document
}

func DecodeJSON[T any](t testing.TB, r io.Reader) (v T) {
	t.Helper()

	decoder := json.NewDecoder(r)
	require.NoError(t, decoder.Decode(&v))
	return v
}
