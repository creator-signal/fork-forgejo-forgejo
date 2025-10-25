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
		org3identicon := `<g color="#660000ff"><g><polygon points="12,12 18,15 15,18"></polygon><polygon points="24,12 21,18 18,15"></polygon><polygon points="24,24 18,21 21,18"></polygon><polygon points="12,24 15,18 18,21"></polygon><polygon points="0,9 12,12 6,12"></polygon><polygon points="0,9 3,0 6,0"></polygon><polygon points="24,3 12,0 18,0"></polygon><polygon points="24,6 24,12 12,6"></polygon><polygon points="3,12 0,24 0,18"></polygon><polygon points="6,12 12,12 6,24"></polygon><polygon points="9,36 12,24 12,30"></polygon><polygon points="9,36 0,33 0,30"></polygon><polygon points="12,33 24,36 18,36"></polygon><polygon points="12,30 12,24 24,30"></polygon></g><g><polygon points="12,12 18,15 15,18"></polygon><polygon points="24,12 21,18 18,15"></polygon><polygon points="24,24 18,21 21,18"></polygon><polygon points="12,24 15,18 18,21"></polygon><polygon points="0,9 12,12 6,12"></polygon><polygon points="0,9 3,0 6,0"></polygon><polygon points="24,3 12,0 18,0"></polygon><polygon points="24,6 24,12 12,6"></polygon><polygon points="3,12 0,24 0,18"></polygon><polygon points="6,12 12,12 6,24"></polygon><polygon points="9,36 12,24 12,30"></polygon><polygon points="9,36 0,33 0,30"></polygon><polygon points="12,33 24,36 18,36"></polygon><polygon points="12,30 12,24 24,30"></polygon></g></g>`

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
		org3identicon := `<g color="#993366ff"><g><polygon points="18,12 24,18 18,24 12,18"></polygon><polygon points="12,12 6,12 6,6 12,6"></polygon><polygon points="24,0 24,6 18,0"></polygon><polygon points="15,9 15,12 12,12 12,9"></polygon><polygon points="0,12 6,12 0,18"></polygon><polygon points="9,21 12,21 12,24 9,24"></polygon><polygon points="12,24 12,30 6,30 6,24"></polygon><polygon points="12,36 12,30 18,36"></polygon><polygon points="21,27 21,24 24,24 24,27"></polygon></g><g><polygon points="18,12 24,18 18,24 12,18"></polygon><polygon points="12,12 6,12 6,6 12,6"></polygon><polygon points="24,0 24,6 18,0"></polygon><polygon points="15,9 15,12 12,12 12,9"></polygon><polygon points="0,12 6,12 0,18"></polygon><polygon points="9,21 12,21 12,24 9,24"></polygon><polygon points="12,24 12,30 6,30 6,24"></polygon><polygon points="12,36 12,30 18,36"></polygon><polygon points="21,27 21,24 24,24 24,27"></polygon></g></g>`

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
