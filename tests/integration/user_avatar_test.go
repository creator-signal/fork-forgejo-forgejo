// Copyright 2021 The Gitea Authors. All rights reserved.
// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: GPL-3.0-or-later

package integration

import (
	"bytes"
	"fmt"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	"forgejo.org/modules/avatar"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserAvatar(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the repo3, is an org

	seed := user2.Email
	if len(seed) == 0 {
		seed = user2.Name
	}

	img, err := avatar.RandomImage([]byte(seed))
	if err != nil {
		require.NoError(t, err)
		return
	}

	session := loginUser(t, "user2")

	imgData := &bytes.Buffer{}

	body := &bytes.Buffer{}

	// Setup multi-part
	writer := multipart.NewWriter(body)
	writer.WriteField("source", "local")
	part, err := writer.CreateFormFile("avatar", "avatar-for-testuseravatar.png")
	if err != nil {
		require.NoError(t, err)
		return
	}

	if err := png.Encode(imgData, &img.Raster); err != nil {
		require.NoError(t, err)
		return
	}

	if _, err := io.Copy(part, imgData); err != nil {
		require.NoError(t, err)
		return
	}

	if err := writer.Close(); err != nil {
		require.NoError(t, err)
		return
	}

	req := NewRequestWithBody(t, "POST", "/user/settings/avatar", body)
	req.Header.Add("Content-Type", writer.FormDataContentType())

	session.MakeRequest(t, req, http.StatusSeeOther)

	user2 = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}) // owner of the repo3, is an org

	req = NewRequest(t, "GET", user2.AvatarLinkWithSize(db.DefaultContext, 0))
	_ = session.MakeRequest(t, req, http.StatusOK)

	req = NewRequestf(t, "GET", "/%s.png", user2.Name)
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, fmt.Sprintf("/avatars/%s", user2.Avatar), resp.Header().Get("location"))

	// Can't test if the response matches because the image is re-generated on upload but checking that this at least doesn't give a 404 should be enough.
}

func TestAvatarAnchorDestination(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	// If the user is logged in, and looking at their own profile,
	// the avatar becomes a link towards the user settings page.
	// Test that the link does not show up when not viewing one's own profile,
	// and that, if the link does show up, there is a corresponding element
	// on the user settings page matching the fragment of the anchor.

	t.Run("viewing other's profile", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		profilePage := NewHTMLParser(t, MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK).Body)
		profilePage.AssertElement(t, "#profile-avatar", true)
		// When viewing another user's profile, there shouldn't be a link to user settings
		profilePage.AssertElement(t, "#profile-avatar a", false)
	})

	t.Run("viewing own profile", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		session := loginUser(t, "user2")

		profilePage := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", "/user2"), http.StatusOK).Body)
		profilePage.AssertElement(t, "#profile-avatar a", true)
		href, has := profilePage.Find("#profile-avatar a").Attr("href")
		assert.True(t, has)

		settingsURL, err := url.Parse(href)
		require.NoError(t, err, "Change avatar link can't be parsed to URL")

		settingsPage := NewHTMLParser(t, session.MakeRequest(t, NewRequest(t, "GET", href), http.StatusOK).Body)
		settingsPage.AssertElement(t, fmt.Sprintf("#%s", settingsURL.Fragment), true)
	})
}

func TestIdenticons(t *testing.T) {
	defer tests.PrepareTestEnv(t)()
	t.Run("User avatar", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		user20 := loginUser(t, "user20")

		avatarSelDefault := "#profile-avatar img.avatar[src='/assets/img/avatar_default.png']"
		avatarSelSvgIdcon := "#profile-avatar svg.avatar.identicon[width='256'][height='256']"
		// ^ specificality of this selector also verifies attributes set by AvatarHTML

		// This is assumed to be a correctly rendered identicon
		org3identicon := `<g color="#660000ff"><g><path d="M24 24 L36 30 L30 36 Z"></path><path d="M48 24 L42 36 L36 30 Z"></path><path d="M48 48 L36 42 L42 36 Z"></path><path d="M24 48 L30 36 L36 42 Z"></path><path d="M0 18 L24 24 L12 24 Z"></path><path d="M0 18 L6 0 L12 0 Z"></path><path d="M48 6 L24 0 L36 0 Z"></path><path d="M48 12 L48 24 L24 12 Z"></path><path d="M6 24 L0 48 L0 36 Z"></path><path d="M12 24 L24 24 L12 48 Z"></path><path d="M18 72 L24 48 L24 60 Z"></path><path d="M18 72 L0 66 L0 60 Z"></path><path d="M24 66 L48 72 L36 72 Z"></path><path d="M24 60 L24 48 L48 60 Z"></path></g><g><path d="M24 24 L36 30 L30 36 Z"></path><path d="M48 24 L42 36 L36 30 Z"></path><path d="M48 48 L36 42 L42 36 Z"></path><path d="M24 48 L30 36 L36 42 Z"></path><path d="M0 18 L24 24 L12 24 Z"></path><path d="M0 18 L6 0 L12 0 Z"></path><path d="M48 6 L24 0 L36 0 Z"></path><path d="M48 12 L48 24 L24 12 Z"></path><path d="M6 24 L0 48 L0 36 Z"></path><path d="M12 24 L24 24 L12 48 Z"></path><path d="M18 72 L24 48 L24 60 Z"></path><path d="M18 72 L0 66 L0 60 Z"></path><path d="M24 66 L48 72 L36 72 Z"></path><path d="M24 60 L24 48 L48 60 Z"></path></g></g>`

		// When going to /user20 for the first time it is expected that the org will have the default avatar
		page := NewHTMLParser(t, user20.MakeRequest(t, NewRequest(t, "GET", "/user20"), http.StatusOK).Body)
		page.AssertElement(t, avatarSelDefault, true)
		page.AssertElement(t, avatarSelSvgIdcon, false)

		// Request deletion of current avatar - it will cause a new one to be generated
		user20.MakeRequest(t, NewRequestWithValues(t, "POST", "/user/settings/avatar/delete", map[string]string{
			"_csrf": GetCSRF(t, user20, "/user/settings"),
		}), http.StatusOK)

		page = NewHTMLParser(t, user20.MakeRequest(t, NewRequest(t, "GET", "/user20"), http.StatusOK).Body)
		page.AssertElement(t, avatarSelDefault, false)

		// Verify SVG contents
		html, _ := page.Find(avatarSelSvgIdcon).Html()
		assert.Equal(t, org3identicon, html)
	})
	t.Run("Org avatar", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		user2 := loginUser(t, "user2") // Has owner rights in org3

		avatarSelDefault := "img.org-avatar.avatar[src='/assets/img/avatar_default.png']"
		avatarSelSvgIdcon := `svg.org-avatar.avatar.identicon[width='100'][height='100']`
		// ^ specificality of this selector also verifies attributes set by AvatarHTML

		// This is assumed to be a correctly rendered identicon
		org3identicon := `<g color="#993366ff"><g><path d="M36 24 L48 36 L36 48 L24 36 Z"></path><path d="M24 24 L12 24 L12 12 L24 12 Z"></path><path d="M48 0 L48 12 L36 0 Z"></path><path d="M30 18 L30 24 L24 24 L24 18 Z"></path><path d="M0 24 L12 24 L0 36 Z"></path><path d="M18 42 L24 42 L24 48 L18 48 Z"></path><path d="M24 48 L24 60 L12 60 L12 48 Z"></path><path d="M24 72 L24 60 L36 72 Z"></path><path d="M42 54 L42 48 L48 48 L48 54 Z"></path></g><g><path d="M36 24 L48 36 L36 48 L24 36 Z"></path><path d="M24 24 L12 24 L12 12 L24 12 Z"></path><path d="M48 0 L48 12 L36 0 Z"></path><path d="M30 18 L30 24 L24 24 L24 18 Z"></path><path d="M0 24 L12 24 L0 36 Z"></path><path d="M18 42 L24 42 L24 48 L18 48 Z"></path><path d="M24 48 L24 60 L12 60 L12 48 Z"></path><path d="M24 72 L24 60 L36 72 Z"></path><path d="M42 54 L42 48 L48 48 L48 54 Z"></path></g></g>`

		// When going to /org3 for the first time it is expected that the org will have the default avatar
		page := NewHTMLParser(t, user2.MakeRequest(t, NewRequest(t, "GET", "/org3"), http.StatusOK).Body)
		page.AssertElement(t, avatarSelDefault, true)
		page.AssertElement(t, avatarSelSvgIdcon, false)

		// Request deletion of current avatar - it will cause a new one to be generated
		user2.MakeRequest(t, NewRequestWithValues(t, "POST", "/org/org3/settings/avatar/delete", map[string]string{
			"_csrf": GetCSRF(t, user2, "/org/org3/settings"),
		}), http.StatusOK)

		page = NewHTMLParser(t, user2.MakeRequest(t, NewRequest(t, "GET", "/org3"), http.StatusOK).Body)
		page.AssertElement(t, avatarSelDefault, false)

		// Verify SVG contents
		html, _ := page.Find(avatarSelSvgIdcon).Html()
		assert.Equal(t, org3identicon, html)
	})
}
