// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"testing"

	"forgejo.org/modules/setting"
	"forgejo.org/modules/test"
	"forgejo.org/routers"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
)

func TestAdminModerationViewReports(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	t.Run("Moderation enabled", func(t *testing.T) {
		defer test.MockVariableValue(&setting.Moderation.Enabled, true)()
		defer test.MockVariableValue(&testWebRoutes, routers.NormalRoutes())()

		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/admin/moderation/reports")
			MakeRequest(t, req, http.StatusSeeOther)
		})

		t.Run("Normal user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user2")
			req := NewRequest(t, "GET", "/admin/moderation/reports")
			session.MakeRequest(t, req, http.StatusForbidden)
		})

		t.Run("Admin user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			assert.Equal(t, true, setting.Moderation.Enabled)

			session := loginUser(t, "user1")
			req := NewRequest(t, "GET", "/admin/moderation/reports")
			session.MakeRequest(t, req, http.StatusOK)
		})
	})

	t.Run("Moderation disabled", func(t *testing.T) {
		t.Run("Anonymous user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequest(t, "GET", "/admin/moderation/reports")
			MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("Normal user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user2")
			req := NewRequest(t, "GET", "/admin/moderation/reports")
			session.MakeRequest(t, req, http.StatusNotFound)
		})

		t.Run("Admin user", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			session := loginUser(t, "user1")
			req := NewRequest(t, "GET", "/admin/moderation/reports")
			session.MakeRequest(t, req, http.StatusNotFound)
		})
	})
}
