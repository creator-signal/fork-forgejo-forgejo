// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web"
	"forgejo.org/tests"
)

// Cancelling the request makes the authentication lookup fail the way it does for a client that hung up.
func TestPackageContainerCanceledRequest(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	canceledRequest := func(t *testing.T, url string, auth func(*RequestWrapper)) *RequestWrapper {
		t.Helper()
		req := NewRequest(t, "GET", url)
		auth(req)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		req.Request = req.Request.WithContext(ctx)
		return req
	}

	t.Run("basic auth", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := canceledRequest(t, fmt.Sprintf("%sv2", setting.AppURL), func(r *RequestWrapper) {
			r.AddBasicAuth(user.Name)
		})

		MakeRequest(t, req, web.StatusClientClosedRequest)
	})

	t.Run("token endpoint", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := canceledRequest(t, fmt.Sprintf("%sv2/token", setting.AppURL), func(r *RequestWrapper) {
			r.AddBasicAuth(user.Name)
		})

		MakeRequest(t, req, web.StatusClientClosedRequest)
	})

	t.Run("uncanceled request still succeeds", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%sv2", setting.AppURL))
		req.AddBasicAuth(user.Name)

		MakeRequest(t, req, http.StatusOK)
	})
}
