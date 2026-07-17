// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

// TestOrgSettingsRepos verifies the end-to-end rendering of the organization
// settings repositories page: a sortable table listing the organization's
// repositories with their git and LFS size columns, reachable only by owners.
func TestOrgSettingsRepos(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// user2 is in the owners team of org3.
	session := loginUser(t, "user2")

	req := NewRequest(t, "GET", "/org/org3/settings/repos")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	// The page must be linked from the settings navigation.
	assert.Positive(t, htmlDoc.doc.Find(".flex-container-nav a[href='/org/org3/settings/repos']").Length(),
		"the settings navigation must link the repositories page")

	// The table headers (name / size / lfs size) must be present.
	assert.Equal(t, 3, htmlDoc.doc.Find("table thead tr th").Length())

	// A known repository owned by org3 must be linked in the table body, and
	// its row must carry the name and the two size columns.
	row := htmlDoc.doc.Find("table tbody tr td a[href='/org3/repo3']").Closest("tr")
	assert.Equal(t, 1, row.Length(), "repo3 must be listed for org3")
	assert.Equal(t, 3, row.Find("td").Length())

	// user4 is a member of org3 but not an owner: the page must be denied.
	memberSession := loginUser(t, "user4")
	memberSession.MakeRequest(t, NewRequest(t, "GET", "/org/org3/settings/repos"), http.StatusNotFound)
}

// TestOrgSettingsReposEmptyState verifies that an organization without any
// repository renders the empty-state message instead of a list of rows.
func TestOrgSettingsReposEmptyState(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// org25 owns no repositories; user1 is an instance admin and therefore has
	// owner access to every organization.
	session := loginUser(t, "user1")

	req := NewRequest(t, "GET", "/org/org25/settings/repos")
	resp := session.MakeRequest(t, req, http.StatusOK)

	assert.Contains(t, resp.Body.String(), "You do not own any repositories.")
}
