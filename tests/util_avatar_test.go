// Copyright 2024 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvatarHTML(t *testing.T) {
	t.Run("contains expected attributes", func(t *testing.T) {
		result := string(AvatarHTML("https://example.com/avatar.png", 32, "avatar-class", "John"))

		assert.Contains(t, result, `src="https://example.com/avatar.png"`)
		assert.Contains(t, result, `width="32"`)
		assert.Contains(t, result, `height="32"`)
		assert.Contains(t, result, `class="avatar-class"`)
		assert.Contains(t, result, `title="John"`)
		assert.Contains(t, result, `loading="lazy"`)
	})

	t.Run("falls back to 'avatar' when name is empty", func(t *testing.T) {
		result := string(AvatarHTML("https://example.com/avatar.png", 16, "avatar", ""))

		assert.Contains(t, result, `title="avatar"`)
	})

	t.Run("escapes HTML in name", func(t *testing.T) {
		result := string(AvatarHTML("https://example.com/avatar.png", 16, "avatar", `<script>alert("xss")</script>`))

		assert.NotContains(t, result, "<script>")
		assert.Contains(t, result, "&lt;script&gt;")
	})

	t.Run("renders as self-closing img tag", func(t *testing.T) {
		result := string(AvatarHTML("https://example.com/avatar.png", 16, "avatar", "name"))

		assert.True(t, strings.HasPrefix(result, "<img "))
		assert.True(t, strings.HasSuffix(result, "/>"))
	})
}
