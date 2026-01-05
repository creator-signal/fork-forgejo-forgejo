// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package templates

import (
	"context"
	"encoding/hex"
	"fmt"
	"html"
	"html/template"

	activities_model "forgejo.org/models/activities"
	"forgejo.org/models/avatars"
	"forgejo.org/models/organization"
	repo_model "forgejo.org/models/repo"
	user_model "forgejo.org/models/user"
	gitea_html "forgejo.org/modules/html"
	"forgejo.org/modules/setting"
)

type AvatarUtils struct {
	ctx context.Context
}

func NewAvatarUtils(ctx context.Context) *AvatarUtils {
	return &AvatarUtils{ctx: ctx}
}

// AvatarHTML creates the HTML for an avatar
func AvatarHTML(srcURL string, size int, class, name string) template.HTML {
	sizeStr := fmt.Sprintf(`%d`, size)

	return template.HTML(`<img loading="lazy" alt="" class="` + class + `" src="` + srcURL + `" title="` + html.EscapeString(name) + `" width="` + sizeStr + `" height="` + sizeStr + `"/>`)
}

// AvatarHTMLSVG creates the HTML for an SVG avatar
func AvatarHTMLSVG(srcURL string, size int, class, name string) template.HTML {
	sizeStr := fmt.Sprintf(`%d`, size)

	return template.HTML(`<img loading="lazy" alt="" class="svg identicon ` + class + `" src="` + srcURL + `" title="` + html.EscapeString(name) + `" width="` + sizeStr + `" height="` + sizeStr + `"/>`)
}

// SVGAvatarURL constructs complete URL for SVG avatar from it's hash
func SVGAvatarURL(hash []byte) string {
	return setting.AppSubURL + "/svg-avatars/" + hex.EncodeToString(hash) + ".svg"
}

// Avatar renders user avatars. args: user, size (int), class (string)
func (au *AvatarUtils) Avatar(item any, others ...any) template.HTML {
	size, class := gitea_html.ParseSizeAndClass(avatars.DefaultAvatarPixelSize, avatars.DefaultAvatarClass, others...)

	var vectorHash []byte
	var rasterURL string
	var displayName string

	switch t := item.(type) {
	case *user_model.User:
		vectorHash = t.AvatarSVGHash
		rasterURL = t.AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	case *repo_model.Collaborator:
		vectorHash = t.AvatarSVGHash
		rasterURL = t.AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	case *organization.Organization:
		vectorHash = t.AsUser().AvatarSVGHash
		rasterURL = t.AsUser().AvatarLinkWithSize(au.ctx, size*setting.Avatar.RenderedSizeFactor)
		displayName = t.DisplayName()
	}

	// Fallback for displayName
	if displayName == "" {
		displayName = "avatar"
	}

	// Try vector avatar first - if it is unwanted it wouldn't be present in the database
	if vectorHash != nil {
		return AvatarHTMLSVG(SVGAvatarURL(vectorHash), size, class, displayName)
	}

	// Fall back to raster avatar if present
	if rasterURL != "" {
		return AvatarHTML(rasterURL, size, class, displayName)
	}

	// Fall back to default avatar URL
	return AvatarHTML(avatars.DefaultAvatarLink(), size, class, "")
}

// AvatarByAction renders user avatars from action. args: action, size (int), class (string)
func (au *AvatarUtils) AvatarByAction(action *activities_model.Action, others ...any) template.HTML {
	action.LoadActUser(au.ctx)
	return au.Avatar(action.ActUser, others...)
}

// AvatarByEmail renders avatars by email address. args: email, name, size (int), class (string)
func (au *AvatarUtils) AvatarByEmail(email, name string, others ...any) template.HTML {
	size, class := gitea_html.ParseSizeAndClass(avatars.DefaultAvatarPixelSize, avatars.DefaultAvatarClass, others...)
	src := avatars.GenerateEmailAvatarFastLink(au.ctx, email, size*setting.Avatar.RenderedSizeFactor)

	if src != "" {
		return AvatarHTML(src, size, class, name)
	}

	return ""
}
