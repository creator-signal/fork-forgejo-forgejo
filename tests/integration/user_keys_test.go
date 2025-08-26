// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestVerifySSHkeyPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 has an SSH key in fixtures to test this on
	session := loginUser(t, "user2")

	page := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/keys"), http.StatusOK).Body)
	link, exists := page.Find("#keys-ssh a.button[href^='?verify_ssh=']").Attr("href")
	assert.True(t, exists)

	page = NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", fmt.Sprintf("/user/settings/keys%s", link)), http.StatusOK).Body)

	// QueryUnescape the link for selector matching
	link, err := url.QueryUnescape(link)
	assert.Nil(t, err)

	// The hint contains a link to the same page the user is at now to get it reloaded if followed
	page.AssertElement(t, fmt.Sprintf("#keys-ssh form[action='/user/settings/keys'] .help a[href='%s']", link), true)
}
