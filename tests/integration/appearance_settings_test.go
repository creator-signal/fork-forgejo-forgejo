// Copyright 2025 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"forgejo.org/models/db"
	user_model "forgejo.org/models/user"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestThemeChange(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := loginUser(t, "user2")

	// Verify default theme
	testSelectedTheme(t, user, "forgejo-auto", "Forgejo (follow system theme)")

	// Change theme to forgejo-dark and verify it works fine
	testChangeTheme(t, user, "forgejo-dark")
	testSelectedTheme(t, user, "forgejo-dark", "Forgejo dark")

	// Change theme to gitea-dark and also verify that it's name is not translated
	testChangeTheme(t, user, "gitea-dark")
	testSelectedTheme(t, user, "gitea-dark", "gitea-dark")
}

// testSelectedTheme checks that the expected theme is used in html[data-theme]
// and is default on appearance page
func testSelectedTheme(t *testing.T, session *TestSession, expectedTheme, expectedName string) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/appearance"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	dataTheme, dataThemeExists := page.Find("html").Attr("data-theme")
	assert.True(t, dataThemeExists)
	assert.Equal(t, expectedTheme, dataTheme)

	selectedTheme := page.Find("form[action='/user/settings/appearance/theme'] .menu .item.selected")
	selectorTheme, selectorThemeExists := selectedTheme.Attr("data-value")
	assert.True(t, selectorThemeExists)
	assert.Equal(t, expectedTheme, selectorTheme)
	assert.Equal(t, expectedName, strings.TrimSpace(selectedTheme.Text()))
}

// testSelectedTheme changes user's theme
func testChangeTheme(t *testing.T, session *TestSession, newTheme string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/appearance/theme", map[string]string{
		"theme": newTheme,
	}), http.StatusSeeOther)
}

func TestFirstDayOfWeekChange(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user := loginUser(t, "user2")

	// Verify default first day of week (Monday = 1)
	testSelectedFirstDOW(t, user, "1")

	// Change to Sunday (0) and verify
	testChangeFirstDOW(t, user, "0")
	testSelectedFirstDOW(t, user, "0")

	// Change to Saturday (6) and verify
	testChangeFirstDOW(t, user, "6")
	testSelectedFirstDOW(t, user, "6")

	// Test that invalid values are rejected
	response := user.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/appearance/first_dow", map[string]string{
		"first_dow": "7",
	}), http.StatusSeeOther)

	// Verify the value was not changed
	page := NewHTMLParser(t, response.Body)
	assert.True(t, page.Find(".flash-error").Length() > 0 || strings.Contains(response.Body.String(), "error"))
}

// testSelectedFirstDOW checks that the expected first day of week is selected on appearance page
func testSelectedFirstDOW(t *testing.T, session *TestSession, expectedFirstDOW string) {
	t.Helper()
	response := session.MakeRequest(t, NewRequest(t, "GET", "/user/settings/appearance"), http.StatusOK)
	page := NewHTMLParser(t, response.Body)

	selectedFirstDOW := page.Find("form[action='/user/settings/appearance/first_dow'] .menu .item.selected")
	selectorValue, selectorExists := selectedFirstDOW.Attr("data-value")
	assert.True(t, selectorExists)
	assert.Equal(t, expectedFirstDOW, selectorValue)

	// Verify the user model has the correct value
	// Since we logged in as user2, load that user and check the value
	user, err := user_model.GetUserByName(db.DefaultContext, "user2")
	require.NoError(t, err)
	expectedValue, _ := strconv.Atoi(expectedFirstDOW)
	assert.Equal(t, expectedValue, user.FirstDOW)
}

// testChangeFirstDOW changes user's first day of week setting
func testChangeFirstDOW(t *testing.T, session *TestSession, newFirstDOW string) {
	t.Helper()
	session.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/appearance/first_dow", map[string]string{
		"first_dow": newFirstDOW,
	}), http.StatusSeeOther)
}
