// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// GetCSRF returns the CSRF token from the given URL using the provided session.
func GetCSRF(t testing.TB, session *TestSession, urlStr string) string {
	t.Helper()
	req := NewRequest(t, "GET", urlStr)
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// Try to find in meta tag
	csrf, exists := htmlDoc.doc.Find("meta[name='_csrf']").Attr("content")
	if exists {
		return csrf
	}

	// Try to find in input field
	csrf, exists = htmlDoc.doc.Find("input[name='_csrf']").Attr("value")
	if exists {
		return csrf
	}

	assert.Fail(t, "CSRF token not found in "+urlStr)
	return ""
}
