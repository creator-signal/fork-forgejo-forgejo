// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"
)

func TestAdminImpersonate(t *testing.T) {
	session := loginUser(t, "user1")

	req := NewRequest(t, "POST", "/admin/users/2/impersonate")
	session.MakeRequest(t, req, http.StatusOK)

	// view user2's private repo
	req = NewRequest(t, "GET", "/user2/repo2")
	session.MakeRequest(t, req, http.StatusOK)

	// Stop impersonating
	req = NewRequest(t, "POST", "/user/stop_impersonating")
	session.MakeRequest(t, req, http.StatusOK)

	// user3 is not a real user, but an organization. should fail
	req = NewRequest(t, "POST", "/admin/users/3/impersonate")
	session.MakeRequest(t, req, http.StatusBadRequest)

	// user1 can't impersonate himself
	req = NewRequest(t, "POST", "/admin/users/1/impersonate")
	session.MakeRequest(t, req, http.StatusBadRequest)

	// you can't impersonate a user that doesn't exist
	req = NewRequest(t, "POST", "/admin/users/9999/impersonate")
	session.MakeRequest(t, req, http.StatusBadRequest)

	// impersonate user4
	req = NewRequest(t, "POST", "/admin/users/4/impersonate")
	session.MakeRequest(t, req, http.StatusOK)

	// try to view user2's private repo, should fail
	req = NewRequest(t, "GET", "/user2/repo2")
	session.MakeRequest(t, req, http.StatusNotFound)
}

// Check that non-admin users aren't allowed to impersonate
// and can't use the /user/stop_impersonating route
func TestNonAdminImpersonate(t *testing.T) {
	session := loginUser(t, "user2")

	// stop_impersonating route should fail
	req := NewRequest(t, "POST", "/user/stop_impersonating")
	session.MakeRequest(t, req, http.StatusBadRequest)

	// user2 shouldn't be able to impersonate anyone
	req = NewRequest(t, "POST", "/admin/users/4/impersonate/")
	session.MakeRequest(t, req, http.StatusForbidden)

	// We should be able to see out own repo, because impersonation failed
	req = NewRequest(t, "GET", "/user2/repo2")
	session.MakeRequest(t, req, http.StatusOK)

	// stop_impersonating route should also fail after a failed
	// impersonation request
	req = NewRequest(t, "POST", "/user/stop_impersonating")
	session.MakeRequest(t, req, http.StatusBadRequest)

	// After all this, user2 should be able to see his own repo
	req = NewRequest(t, "GET", "/user2/repo2")
	session.MakeRequest(t, req, http.StatusOK)
}
