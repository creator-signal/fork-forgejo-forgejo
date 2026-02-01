// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"
)

func TestOrgMembersPage(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	testPage := "/org/org3/members"

	t.Run("Guest PoV", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		doc := NewHTMLParser(t, MakeRequest(t, NewRequest(t, "GET", testPage), http.StatusOK).Body)
		/* No interactive buttons - though such evaluation is easy to break in rename */
		doc.AssertElement(t, ".members .list .link-action", false)
		doc.AssertElement(t, ".members .list .delete-button", false)

	})
}
