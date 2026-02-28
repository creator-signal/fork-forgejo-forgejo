// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	packages_model "forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	packages_module "forgejo.org/modules/packages"
	packages_service "forgejo.org/services/packages"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPackageWebViewSlashInName tests that package view and settings pages work correctly
// when the package name contains a forward slash (e.g. scoped npm packages, container images).
// Regression test for https://codeberg.org/forgejo/forgejo/issues/7034
func TestPackageWebViewSlashInName(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageName := "test/package"
	packageVersion := "1.0.0"

	// Create the package directly via the service layer to bypass API name validation,
	// since slashes are valid in names for many package types (container, npm scoped, etc.).
	buf, err := packages_module.CreateHashedBufferFromReader(strings.NewReader("test content"))
	require.NoError(t, err)
	defer buf.Close()

	ctx := t.Context()
	_, _, err = packages_service.CreatePackageAndAddFile(
		ctx,
		&packages_service.PackageCreationInfo{
			PackageInfo: packages_service.PackageInfo{
				Owner:       user,
				PackageType: packages_model.TypeGeneric,
				Name:        packageName,
				Version:     packageVersion,
			},
			Creator: user,
		},
		&packages_service.PackageFileCreationInfo{
			PackageFileInfo: packages_service.PackageFileInfo{
				Filename: "file.bin",
			},
			Creator: user,
			Data:    buf,
			IsLead:  true,
		},
	)
	require.NoError(t, err)

	session := loginUser(t, user.Name)
	encodedName := url.PathEscape(packageName)
	viewURL := fmt.Sprintf("/%s/-/packages/generic/%s/%s", user.Name, encodedName, packageVersion)
	settingsURL := viewURL + "/settings"

	t.Run("ViewPageSettingsLink", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", viewURL)
		resp := session.MakeRequest(t, req, http.StatusOK)

		body := resp.Body.String()
		// The settings link must use the correctly-encoded URL (not double-encoded %252F)
		assert.Contains(t, body, fmt.Sprintf(`href="%s"`, settingsURL),
			"settings link should use correctly-encoded URL with %%2F, not %%252F")
		assert.NotContains(t, body, "%252F",
			"settings link must not contain double-encoded slash %%252F")
	})

	t.Run("SettingsPageAccessible", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", settingsURL)
		resp := session.MakeRequest(t, req, http.StatusOK)

		body := resp.Body.String()
		// The form actions must use the correctly-encoded URL (not double-encoded %252F)
		assert.Contains(t, body, fmt.Sprintf(`action="%s"`, settingsURL),
			"form action should use correctly-encoded URL with %%2F, not %%252F")
		assert.NotContains(t, body, "%252F",
			"form action must not contain double-encoded slash %%252F")
	})
}
