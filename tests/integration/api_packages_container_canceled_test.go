// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	stdctx "context"
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	app_context "forgejo.org/services/context"
	"forgejo.org/tests"
)

// A client that abandons a request, which `docker push` does routinely with the many concurrent blob HEAD requests it
// opens, must not be answered with an internal server error. The authentication lookup runs against the request
// context, so cancelling it makes that lookup fail the same way it does in production.
// See https://codeberg.org/forgejo/forgejo/issues/13782
func TestPackageContainerCanceledRequest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	canceledRequest := func(t *testing.T, url string, auth func(*RequestWrapper)) *RequestWrapper {
		t.Helper()
		req := NewRequest(t, "GET", url)
		auth(req)
		ctx, cancel := stdctx.WithCancel(t.Context())
		cancel()
		req.Request = req.Request.WithContext(ctx)
		return req
	}

	t.Run("basic auth", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := canceledRequest(t, fmt.Sprintf("%sv2", setting.AppURL), func(r *RequestWrapper) {
			r.AddBasicAuth(user.Name)
		})

		MakeRequest(t, req, app_context.StatusClientClosedRequest)
	})

	// The token endpoint is what a container client authenticates against first, and it is served by the other
	// router that had to be fixed.
	t.Run("token endpoint", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := canceledRequest(t, fmt.Sprintf("%sv2/token", setting.AppURL), func(r *RequestWrapper) {
			r.AddBasicAuth(user.Name)
		})

		MakeRequest(t, req, app_context.StatusClientClosedRequest)
	})

	t.Run("uncanceled request still succeeds", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL))
		req.AddBasicAuth(user.Name)

		MakeRequest(t, req, http.StatusOK)
	})
}
